package node

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/mosaicnetworks/babble/src/config"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	_state "github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

// Node defines a babble node
type Node struct {
	// The node is implemented as a state-machine. The embedded state Manager
	// object is used to manage the node's state.
	_state.Manager

	conf *config.Config

	logger *logrus.Entry

	// core is the link between the node and the underlying hashgraph. It
	// controls some higher-level operations like inserting a list of events,
	// keeping track of the peers list, fast-forwarding, etc.
	core     *Core
	coreLock sync.Mutex

	// transport is the object used to transmit and receive commands to other
	// nodes.
	trans net.Transport
	netCh <-chan net.RPC

	// proxy is the link between the node and the application. It is used to
	// commit blocks from Babble to the application, and relay submitted
	// transactions from the application to Babble.
	proxy proxy.AppProxy

	// submitCh is where the node listens for incoming transactions to be
	// submitted to Babble
	submitCh chan []byte

	// sigCh is where the node listens for signals to politely leave the Babble
	// network. It listens to SIGINT and SIGTERM
	sigCh chan os.Signal

	// shutdownCh is where the node listens for commands to cleanly shutdown.
	shutdownCh chan struct{}

	// suspendCh is used to signal the node to enter the Suspended state.
	suspendCh chan struct{}

	// The node runs the controlTimer in the background to periodically receive
	// signals to initiate gossip routines. It is paused, reset, etc., based on
	// the node's current state
	controlTimer *ControlTimer

	start        time.Time
	syncRequests int
	syncErrors   int

	// initialUndeterminedEvents keeps a record of how many undetermined events
	// there were upon initalizing the node. This value is regularly compared
	// to a current number of undetermined events and the SuspendLimit to
	// suspend the node.
	initialUndeterminedEvents int
}

// NewNode is a factory method that returns a Node instance
func NewNode(conf *config.Config,
	validator *Validator,
	peers *peers.PeerSet,
	genesisPeers *peers.PeerSet,
	store hg.Store,
	trans net.Transport,
	proxy proxy.AppProxy,
) *Node {

	//Prepare sigCh to relay SIGINT and SIGTERM system calls
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	core := NewCore(validator,
		peers,
		genesisPeers,
		store,
		proxy.CommitBlock,
		conf.MaintenanceMode,
		conf.Logger())

	node := Node{
		conf:         conf,
		logger:       conf.Logger(),
		core:         core,
		trans:        trans,
		netCh:        trans.Consumer(),
		proxy:        proxy,
		submitCh:     proxy.SubmitCh(),
		sigCh:        sigCh,
		shutdownCh:   make(chan struct{}),
		suspendCh:    make(chan struct{}),
		controlTimer: NewRandomControlTimer(),
	}

	return &node
}

/*******************************************************************************
Public Methods
*******************************************************************************/

// Init controls the bootstrap process and sets the node's initial state based
// on configuration (Babbling, CatchingUp, Joining, or Suspended).
func (n *Node) Init() error {

	// if the bootstrap option is set, load the hashgraph from an existing
	// database (if bootstrap option is set in config).
	if n.conf.Bootstrap {
		n.logger.Debug("Bootstrap")
		if err := n.core.Bootstrap(); err != nil {
			return err
		}
		n.logger.Debug("Bootstrap completed")
	}

	// if the maintenance-mode option is not enabled, open the network transport
	// and decide wether to babble normally, fast-forward, or join. Otherwise
	// enter the suspended state.
	if !n.conf.MaintenanceMode {
		n.logger.Debug("Start Listening")
		go n.trans.Listen()

		_, ok := n.core.peers.ByID[n.core.validator.ID()]
		if ok {
			n.logger.Debug("Node belongs to PeerSet")
			n.setBabblingOrCatchingUpState()
		} else {
			n.logger.Debug("Node does not belong to PeerSet => Joining")
			n.transition(_state.Joining)
		}
	} else {
		n.transition(_state.Suspended)
	}

	// record the number of initial undetermined events so as to suspend the
	// node when undetermined events exceed this value by SuspendLimit.
	n.initialUndeterminedEvents = len(n.core.GetUndeterminedEvents())

	return nil
}

// Run invokes the main loop of the node. The gossip parameter controls whether
// to actively participate in gossip or not.
func (n *Node) Run(gossip bool) {
	// The ControlTimer allows the background routines to control the heartbeat
	// timer when the node is in the Babbling state. The timer should only be
	// running when there are uncommitted transactions in the system.
	go n.controlTimer.Run(n.conf.HeartbeatTimeout)

	// Execute some background work regardless of the state of the node.
	go n.doBackgroundWork()

	// Execute Node State Machine
	for {
		// Run different routines depending on node state
		state := n.GetState()

		switch state {
		case _state.Babbling:
			n.babble(gossip)
		case _state.CatchingUp:
			n.fastForward()
		case _state.Joining:
			n.join()
		case _state.Suspended:
			time.Sleep(2000 * time.Millisecond)
		case _state.Shutdown:
			return
		}
	}
}

// RunAsync runs the node in a separate goroutine
func (n *Node) RunAsync(gossip bool) {
	n.logger.WithField("gossip", gossip).Debug("runasync")
	go n.Run(gossip)
}

// Leave causes the node to politely leave the network via a LeaveRequest and
// wait for the node to be removed from the validator-list via consensus.
func (n *Node) Leave() error {
	n.logger.Info("LEAVING")

	defer n.Shutdown()

	err := n.core.Leave(n.conf.JoinTimeout)
	if err != nil {
		n.logger.WithError(err).Error("Leaving")
		return err
	}

	return nil
}

// Shutdown attempts to cleanly shutdown the node by waiting for pending work to
// be finished, stopping the control-timer, and closing the transport.
func (n *Node) Shutdown() {
	if n.GetState() != _state.Shutdown {
		n.logger.Info("SHUTDOWN")

		// exit any non-shutdown state immediately
		n.transition(_state.Shutdown)

		// stop and wait for concurrent operations
		close(n.shutdownCh)
		n.WaitRoutines()

		// close transport and store
		n.trans.Close()

		n.core.hg.Store.Close()
	}
}

// Suspend puts the node in Suspended mode. It doesn't close the transport
// because it needs to respond to incoming SyncRequests.
func (n *Node) Suspend() {
	if n.GetState() != _state.Suspended &&
		n.GetState() != _state.Shutdown {

		n.logger.Info("SUSPEND")

		n.transition(_state.Suspended)

		// Stop and wait for concurrent operations
		close(n.suspendCh)
		n.WaitRoutines()
	}
}

// GetID returns the numeric ID of the node's validator
func (n *Node) GetID() uint32 {
	return n.core.validator.ID()
}

// GetStats returns information about the node.
func (n *Node) GetStats() map[string]string {
	toString := func(i *int) string {
		if i == nil {
			return "nil"
		}

		return strconv.Itoa(*i)
	}

	timeElapsed := time.Since(n.start)

	consensusEvents := n.core.GetConsensusEventsCount()

	consensusEventsPerSecond := float64(consensusEvents) / timeElapsed.Seconds()

	lastConsensusRound := n.core.GetLastConsensusRoundIndex()

	var consensusRoundsPerSecond float64

	if lastConsensusRound != nil {
		consensusRoundsPerSecond = float64(*lastConsensusRound) / timeElapsed.Seconds()
	}

	s := map[string]string{
		"last_consensus_round":   toString(lastConsensusRound),
		"last_block_index":       strconv.Itoa(n.core.GetLastBlockIndex()),
		"consensus_events":       strconv.Itoa(consensusEvents),
		"consensus_transactions": strconv.Itoa(n.core.GetConsensusTransactionsCount()),
		"undetermined_events":    strconv.Itoa(len(n.core.GetUndeterminedEvents())),
		"transaction_pool":       strconv.Itoa(len(n.core.transactionPool)),
		"num_peers":              strconv.Itoa(n.core.peerSelector.Peers().Len()),
		"last_peer_change":       strconv.Itoa(n.core.LastPeerChangeRound),
		"sync_rate":              strconv.FormatFloat(n.syncRate(), 'f', 2, 64),
		"events_per_second":      strconv.FormatFloat(consensusEventsPerSecond, 'f', 2, 64),
		"rounds_per_second":      strconv.FormatFloat(consensusRoundsPerSecond, 'f', 2, 64),
		"round_events":           strconv.Itoa(n.core.GetLastCommitedRoundEventsCount()),
		"id":                     fmt.Sprint(n.core.validator.ID()),
		"state":                  n.GetState().String(),
		"moniker":                n.core.validator.Moniker,
		"time":                   strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	return s
}

// GetBlock returns a block
func (n *Node) GetBlock(blockIndex int) (*hg.Block, error) {
	return n.core.hg.Store.GetBlock(blockIndex)
}

// GetLastBlockIndex returns the index of the last known block
func (n *Node) GetLastBlockIndex() int {
	return n.core.GetLastBlockIndex()
}

// GetLastConsensusRoundIndex returns the index of the last consensus round
func (n *Node) GetLastConsensusRoundIndex() int {
	lcr := n.core.GetLastConsensusRoundIndex()
	if lcr == nil {
		return -1
	}
	return *lcr
}

// GetPeers returns the current peers. Not necessarily equal to the
// current validator-set
func (n *Node) GetPeers() []*peers.Peer {
	return n.core.peers.Peers
}

// GetValidatorSet returns the validator-set associated to a round
func (n *Node) GetValidatorSet(round int) ([]*peers.Peer, error) {
	peerSet, err := n.core.hg.Store.GetPeerSet(round)
	if err != nil {
		return nil, err
	}
	return peerSet.Peers, nil
}

// GetAllValidatorSets returns the entire history of validator-sets
func (n *Node) GetAllValidatorSets() (map[int][]*peers.Peer, error) {
	return n.core.hg.Store.GetAllPeerSets()
}

/*******************************************************************************
Background
*******************************************************************************/

// doBackgroundWork coninuously listens to incoming transactions, and the sigint
// signal, regardless of the node's state. It also listens to incoming gossip.
func (n *Node) doBackgroundWork() {
	for {
		select {
		case rpc := <-n.netCh:
			n.GoFunc(func() {
				n.processRPC(rpc)
				n.resetTimer()
			})
		case t := <-n.submitCh:
			n.logger.Debug("Adding Transaction")
			n.addTransaction(t)
			n.resetTimer()
		case <-n.shutdownCh:
			return
		case s := <-n.sigCh:
			n.logger.Debugf("Process %s - LEAVE", s.String())
			n.Leave()
			os.Exit(0)
		}
	}
}

// resetTimer resets the control timer to the configured hearbeat timeout, or
// slows it down if the node is not busy.
func (n *Node) resetTimer() {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	if !n.controlTimer.set {
		ts := n.conf.HeartbeatTimeout

		//Slow gossip if nothing interesting to say
		if !n.core.Busy() {
			ts = time.Duration(n.conf.SlowHeartbeatTimeout)
		}

		n.controlTimer.resetCh <- ts
	}
}

// checkSuspend suspends the node if the number of undetermined events in the
// hashgraph exceeds initialUndeterminedEvents by n*SuspendLimit (where n is the
// the size of the current validator set), or if the validator has been evicted.
func (n *Node) checkSuspend() {

	// check too many undetermined events
	newUndeterminedEvents := len(n.core.GetUndeterminedEvents()) - n.initialUndeterminedEvents
	tooManyUndeterminedEvents := newUndeterminedEvents > n.conf.SuspendLimit*n.core.validators.Len()

	// check evicted
	evicted := n.core.hg.LastConsensusRound != nil &&
		n.core.RemovedRound > 0 &&
		n.core.RemovedRound > n.core.AcceptedRound &&
		*n.core.hg.LastConsensusRound >= n.core.RemovedRound

	// suspend if too many undetermined events or evicted
	if tooManyUndeterminedEvents || evicted {
		n.logger.WithFields(logrus.Fields{
			"evicted":                   evicted,
			"tooManyUndeterminedEvents": tooManyUndeterminedEvents,
			"id":            n.GetID(),
			"removedRound":  n.core.RemovedRound,
			"acceptedRound": n.core.AcceptedRound,
		}).Debugf("SUSPEND")

		n.Suspend()
	}
}

/*******************************************************************************
Babbling
*******************************************************************************/

// babble periodically initiates gossip or monologue as triggered by the
// controlTimer.
func (n *Node) babble(gossip bool) {
	n.logger.Info("BABBLING")

	for {
		select {
		case <-n.controlTimer.tickCh:
			if gossip {
				peer := n.core.peerSelector.Next()
				if peer != nil {
					n.GoFunc(func() {
						n.gossip(peer)
					})
				} else {
					n.monologue()
				}
			}
			n.resetTimer()
			n.checkSuspend()
		case <-n.suspendCh:
			return
		case <-n.shutdownCh:
			return
		}
	}
}

// monologue is called when the node is alone in the network but wants to record
// some events anyway.
func (n *Node) monologue() error {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	if n.core.Busy() {
		err := n.core.AddSelfEvent("")
		if err != nil {
			n.logger.WithError(err).Error("monologue, AddSelfEvent()")
			return err
		}

		err = n.core.ProcessSigPool()
		if err != nil {
			n.logger.WithError(err).Error("monologue, ProcessSigPool()")
			return err
		}
	}

	return nil
}

// gossip performs a pull-push gossip operation with the selected peer.
func (n *Node) gossip(peer *peers.Peer) error {
	var connected bool

	defer func() {
		// update peer selector
		n.core.selectorLock.Lock()
		newConnection := n.core.peerSelector.UpdateLast(peer.ID(), connected)
		n.core.selectorLock.Unlock()
		if newConnection {
			n.logger.WithFields(logrus.Fields{
				"peer_ID":      peer.ID(),
				"peer_moniker": peer.Moniker,
			}).Info("Connected")
		}
	}()

	// pull
	otherKnownEvents, err := n.pull(peer)
	if err != nil {
		n.logger.WithError(err).Warn("gossip pull")
		return err
	}

	// push
	err = n.push(peer, otherKnownEvents)
	if err != nil {
		n.logger.WithError(err).Warn("gossip push")
		return err
	}

	n.logStats()

	connected = true

	return nil
}

// pull performs a SyncRequest and processes the response.
func (n *Node) pull(peer *peers.Peer) (otherKnownEvents map[uint32]int, err error) {
	//Compute Known
	n.coreLock.Lock()
	knownEvents := n.core.KnownEvents()
	n.coreLock.Unlock()

	//Send SyncRequest
	start := time.Now()
	resp, err := n.requestSync(peer.NetAddr, knownEvents, n.conf.SyncLimit)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestSync()")

	if err != nil {
		n.logger.WithField("error", err).Warn("requestSync()")
		return nil, err
	}

	n.logger.WithFields(logrus.Fields{
		"from_id": resp.FromID,
		"events":  len(resp.Events),
		"known":   resp.Known,
	}).Debug("SyncResponse")

	//Add Events to Hashgraph and create new Head if necessary
	n.coreLock.Lock()
	err = n.sync(peer.ID(), resp.Events)
	n.coreLock.Unlock()

	if err != nil {
		n.logger.WithField("error", err).Error("sync()")
		return nil, err
	}

	return resp.Known, nil
}

// push preforms an EagerSyncRequest
func (n *Node) push(peer *peers.Peer, knownEvents map[uint32]int) error {
	// Compute Diff
	start := time.Now()
	n.coreLock.Lock()
	eventDiff, err := n.core.EventDiff(knownEvents)
	n.coreLock.Unlock()
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Diff()")
	if err != nil {
		n.logger.WithField("error", err).Error("Calculating Diff")
		return err
	}

	if len(eventDiff) > 0 {
		// do not push more than sync_limit events
		if n.conf.SyncLimit < len(eventDiff) {
			n.logger.WithFields(logrus.Fields{
				"sync_limit":  n.conf.SyncLimit,
				"diff_length": len(eventDiff),
			}).Debug("Push sync_limit")
			eventDiff = eventDiff[:n.conf.SyncLimit]
		}

		// Convert to WireEvents
		wireEvents, err := n.core.ToWire(eventDiff)
		if err != nil {
			n.logger.WithField("error", err).Debug("Converting to WireEvent")
			return err
		}

		// Create and Send EagerSyncRequest
		start = time.Now()
		resp2, err := n.requestEagerSync(peer.NetAddr, wireEvents)
		elapsed = time.Since(start)
		n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestEagerSync()")
		if err != nil {
			n.logger.WithField("error", err).Warn("requestEagerSync()")
			return err
		}
		n.logger.WithFields(logrus.Fields{
			"from_id": resp2.FromID,
			"success": resp2.Success,
		}).Debug("EagerSyncResponse")
	}

	return nil
}

// sync attempts to insert a list of events into the hashgraph, record a new
// sync event, and process the signature pool.
func (n *Node) sync(fromID uint32, events []hg.WireEvent) error {
	//Insert Events in Hashgraph and create new Head if necessary
	start := time.Now()
	err := n.core.Sync(fromID, events)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Sync()")
	if err != nil {
		if !hg.IsNormalSelfParentError(err) {
			n.logger.WithError(err).Error()
			return err
		}
	}

	//Process SignaturePool
	start = time.Now()
	err = n.core.ProcessSigPool()
	elapsed = time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("ProcessSigPool()")
	if err != nil {
		n.logger.WithError(err).Error()
		return err
	}

	return nil
}

/*******************************************************************************
CatchingUp
*******************************************************************************/

// fastForward enacts "CatchingUp"
func (n *Node) fastForward() error {
	n.logger.Info("CATCHING-UP")

	//wait until sync routines finish
	n.WaitRoutines()

	var err error

	// loop through all peers to check who is the most ahead, then fast-forward
	// from them. If no-one is ready to fast-forward, transition to the Babbling
	// state.
	resp := n.getBestFastForwardResponse()
	if resp == nil {
		n.logger.Error("getBestFastForwardResponse returned nil => Babbling")
		n.transition(_state.Babbling)
		return fmt.Errorf("getBestFastForwardResponse returned nil")
	}

	//update app from snapshot
	err = n.proxy.Restore(resp.Snapshot)
	if err != nil {
		n.logger.WithError(err).Error("Restoring App from Snapshot")
		return err
	}

	//prepare core. ie: fresh hashgraph
	n.coreLock.Lock()
	err = n.core.FastForward(&resp.Block, &resp.Frame)
	n.coreLock.Unlock()
	if err != nil {
		n.logger.WithError(err).Error("Fast Forwarding Hashgraph")
		return err
	}

	err = n.core.ProcessAcceptedInternalTransactions(resp.Block.RoundReceived(), resp.Block.InternalTransactionReceipts())
	if err != nil {
		n.logger.WithError(err).Error("Processing AnchorBlock InternalTransactionReceipts")
	}

	n.logger.Debug("FastForward OK")

	n.transition(_state.Babbling)

	return nil
}

// getBestFastForwardResponse performs a FastForwardRequest with all known peers
// and only selects the one corresponding to the hightest block number.
func (n *Node) getBestFastForwardResponse() *net.FastForwardResponse {
	var bestResponse *net.FastForwardResponse
	maxBlock := 0

	for _, p := range n.core.peerSelector.Peers().Peers {
		start := time.Now()
		resp, err := n.requestFastForward(p.NetAddr)
		elapsed := time.Since(start)
		n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestFastForward()")
		if err != nil {
			n.logger.WithField("error", err).Error("requestFastForward()")
			continue
		}

		n.logger.WithFields(logrus.Fields{
			"from_id":              resp.FromID,
			"block_index":          resp.Block.Index(),
			"block_round_received": resp.Block.RoundReceived(),
			"frame_events":         len(resp.Frame.Events),
			"frame_roots":          resp.Frame.Roots,
			"frame_peers":          len(resp.Frame.Peers),
			"snapshot":             resp.Snapshot,
		}).Debug("FastForwardResponse")

		if resp.Block.Index() > maxBlock {
			bestResponse = &resp
			maxBlock = resp.Block.Index()
		}
	}

	return bestResponse
}

/*******************************************************************************
Joining
*******************************************************************************/

// join attempts to add the node's validator public-key to the current
// validator-set via an InternalTransaction which has to go through consensus.
func (n *Node) join() error {
	n.logger.Info("JOINING")

	peer := n.core.peerSelector.Next()

	start := time.Now()
	resp, err := n.requestJoin(peer.NetAddr)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestJoin()")

	if err != nil {
		n.logger.Error("Cannot join:", peer.NetAddr, err)
		return err
	}

	n.logger.WithFields(logrus.Fields{
		"from_id":        resp.FromID,
		"accepted":       resp.Accepted,
		"accepted_round": resp.AcceptedRound,
		"peers":          len(resp.Peers),
	}).Debug("JoinResponse")

	if resp.Accepted {
		// Set AcceptedRound, which is the next round at which the node is
		// allowed to create SelfEvents, and reset RemovedRound to "unsuspend" a
		// node if had been evicted prior to rejoining.
		n.core.AcceptedRound = resp.AcceptedRound
		n.core.RemovedRound = -1

		n.setBabblingOrCatchingUpState()
	} else {
		// Then JoinRequest was explicitly refused by the curren peer-set. This
		// is not an error.
		n.logger.Info("JoinRequest rejected")
		n.Shutdown()
	}

	return nil
}

/*******************************************************************************
Utils
*******************************************************************************/

// transition changes the node state and notifies the app via the proxy's
// OnStateChanged callback
func (n *Node) transition(state _state.State) {
	n.SetState(state)

	if err := n.proxy.OnStateChanged(state); err != nil {
		n.logger.Error(err)
	}
}

// setBabblingOrCatchingUpState sets the node's state to CatchingUp if fast-sync
// is enabled, or to Babbling if fast-sync is not enabled.
func (n *Node) setBabblingOrCatchingUpState() {
	if n.conf.EnableFastSync {
		n.logger.Debug("FastSync enabled => CatchingUp")
		n.transition(_state.CatchingUp)
	} else {
		n.logger.Debug("FastSync not enabled => Babbling")
		if err := n.core.SetHeadAndSeq(); err != nil {
			n.core.SetHeadAndSeq()
		}
		n.transition(_state.Babbling)
	}
}

// addTransaction is a thread-safe function to add and incoming transaction to
// the core's transaction-pool.
func (n *Node) addTransaction(tx []byte) {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	n.core.AddTransactions([][]byte{tx})
}

// logStats logs the output returned by GetStats()
func (n *Node) logStats() {
	stats := n.GetStats()

	n.logger.WithFields(logrus.Fields{
		"last_consensus_round":   stats["last_consensus_round"],
		"last_block_index":       stats["last_block_index"],
		"consensus_events":       stats["consensus_events"],
		"consensus_transactions": stats["consensus_transactions"],
		"undetermined_events":    stats["undetermined_events"],
		"transaction_pool":       stats["transaction_pool"],
		"num_peers":              stats["num_peers"],
		"sync_rate":              stats["sync_rate"],
		"events/s":               stats["events_per_second"],
		"rounds/s":               stats["rounds_per_second"],
		"round_events":           stats["round_events"],
		"id":                     stats["id"],
		"state":                  stats["state"],
		"moniker":                stats["moniker"],
	}).Debug("Stats")
}

// syncRate computes the ratio of sync-errors over sync-requests
func (n *Node) syncRate() float64 {
	var syncErrorRate float64

	if n.syncRequests != 0 {
		syncErrorRate = float64(n.syncErrors) / float64(n.syncRequests)
	}

	return 1 - syncErrorRate
}
