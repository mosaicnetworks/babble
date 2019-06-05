package node

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

//Node defines a babble node
type Node struct {
	state

	conf   *Config
	logger *logrus.Entry

	validator *Validator

	core     *Core
	coreLock sync.Mutex

	trans net.Transport
	netCh <-chan net.RPC

	proxy    proxy.AppProxy
	submitCh chan []byte

	sigintCh   chan os.Signal
	shutdownCh chan struct{}

	controlTimer *ControlTimer

	start        time.Time
	syncRequests int
	syncErrors   int
}

//NewNode is a factory method that returns a Node instance
func NewNode(conf *Config,
	validator *Validator,
	peers *peers.PeerSet,
	genesisPeers *peers.PeerSet,
	store hg.Store,
	trans net.Transport,
	proxy proxy.AppProxy,
) *Node {
	//Prepare sigintCh to relay SIGINT system calls
	sigintCh := make(chan os.Signal)
	signal.Notify(sigintCh, os.Interrupt, syscall.SIGINT)

	node := Node{
		validator:    validator,
		conf:         conf,
		logger:       conf.Logger.WithField("this_id", validator.ID()),
		core:         NewCore(validator, peers, genesisPeers, store, proxy.CommitBlock, conf.Logger),
		trans:        trans,
		netCh:        trans.Consumer(),
		proxy:        proxy,
		submitCh:     proxy.SubmitCh(),
		sigintCh:     sigintCh,
		shutdownCh:   make(chan struct{}),
		controlTimer: NewRandomControlTimer(),
	}

	return &node
}

//Init intialises the node
func (n *Node) Init() error {
	if n.conf.Bootstrap {
		n.logger.Debug("Bootstrap")
		if err := n.core.Bootstrap(); err != nil {
			return err
		}
	}

	_, ok := n.core.peers.ByID[n.validator.ID()]
	if ok {
		n.logger.Debug("Node belongs to PeerSet")
		n.babbleOrCatchUp()
	} else {
		n.logger.Debug("Node does not belong to PeerSet => Joining")
		n.setState(Joining)
	}

	return nil
}

func (n *Node) babbleOrCatchUp() {
	if n.conf.EnableFastSync {
		n.logger.Debug("FastSync enabled => CatchingUp")
		n.setState(CatchingUp)
	} else {
		n.logger.Debug("FastSync not enabled => Babbling")
		if err := n.core.SetHeadAndSeq(); err != nil {
			n.core.SetHeadAndSeq()
		}
		n.setState(Babbling)
	}
}

//RunAsync calls Run as a separate thread
func (n *Node) RunAsync(gossip bool) {
	n.logger.WithField("gossip", gossip).Debug("runasync")

	go n.Run(gossip)
}

//Run invokes the main loop of the node
func (n *Node) Run(gossip bool) {
	//The ControlTimer allows the background routines to control the
	//heartbeat timer when the node is in the Babbling state. The timer should
	//only be running when there are uncommitted transactions in the system.
	go n.controlTimer.Run(n.conf.HeartbeatTimeout)

	//Execute some background work regardless of the state of the node.
	go n.doBackgroundWork()

	//Execute Node State Machine
	for {
		//Run different routines depending on node state
		state := n.getState()

		n.logger.WithField("state", state.String()).Debug("Run loop")

		switch state {
		case Babbling:
			n.babble(gossip)
		case CatchingUp:
			n.fastForward()
		case Joining:
			n.join()
		case Shutdown:
			return
		}
	}
}

//ResetTimer
func (n *Node) resetTimer() {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	if !n.controlTimer.set {
		ts := n.conf.HeartbeatTimeout

		//Slow gossip if nothing interesting to say
		if !n.core.Busy() {
			ts = time.Duration(time.Second)
		}

		n.controlTimer.resetCh <- ts
	}
}

func (n *Node) doBackgroundWork() {
	for {
		select {
		//XXX moved from babble()
		case rpc := <-n.netCh:
			n.goFunc(func() {
				n.logger.Debug("Processing RPC")
				n.processRPC(rpc)
				n.resetTimer()
			})
		case t := <-n.submitCh:
			n.logger.Debug("Adding Transaction")
			n.addTransaction(t)
			n.resetTimer()
		case <-n.shutdownCh:
			return
		case <-n.sigintCh:
			n.logger.Debug("Reacting to SIGINT - LEAVE")
			n.Leave()
			os.Exit(0)
		}
	}
}

// babble processes incoming RPC requests and periodically initiates gossip
// while there is something to gossip about.
func (n *Node) babble(gossip bool) {
	n.logger.Debug("BABBLING")

	for {
		select {
		case <-n.controlTimer.tickCh:
			if gossip {
				n.logger.Debug("Time to gossip!")
				peer := n.core.peerSelector.Next()
				if peer != nil {
					n.goFunc(func() { n.gossip(peer) })
				} else {
					n.monologue()
				}
			}
			n.resetTimer()
		case <-n.shutdownCh:
			return
		}
	}
}

//fastForward enacts "CatchingUp"
func (n *Node) fastForward() error {
	n.logger.Debug("CATCHING-UP")

	//wait until sync routines finish
	n.waitRoutines()

	var err error

	// loop through all peers to check who is the most ahead, then fast-forward
	// from them. If no-one is ready good to fast-forward, go the the Babbling
	// state.
	resp := n.getBestFastForwardResponse()
	if resp == nil {
		n.logger.Error("getBestFastForwardResponse returned nil => Babbling")
		n.setState(Babbling)
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

	n.setState(Babbling)

	return nil
}

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

func (n *Node) join() error {
	n.logger.Debug("JOINING")

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
		n.core.AcceptedRound = resp.AcceptedRound
		n.babbleOrCatchUp()
	} else {
		// Then JoinRequest was explicitely refused by the curren peer-set. This
		// is not an error.
		n.logger.Debug("JoinRequest refused. Shutting down.")
		n.Shutdown()
	}

	return nil
}

//Leave causes the node to leave the network
func (n *Node) Leave() error {
	n.logger.Debug("LEAVING")

	defer n.Shutdown()

	err := n.core.Leave(n.conf.JoinTimeout)
	if err != nil {
		n.logger.WithError(err).Error("Leaving")
		return err
	}

	return nil
}

//gossip performs a pull-push gossip operation with the selected peer.
func (n *Node) gossip(peer *peers.Peer) error {
	//pull
	otherKnownEvents, err := n.pull(peer)
	if err != nil {
		n.logger.WithError(err).Error("gossip pull")
		return err
	}

	//push
	err = n.push(peer, otherKnownEvents)
	if err != nil {
		n.logger.WithError(err).Error("gossip push")
		return err
	}

	//update peer selector
	n.core.selectorLock.Lock()
	n.core.peerSelector.UpdateLast(peer.ID())
	n.core.selectorLock.Unlock()

	n.logStats()

	return nil
}

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
		n.logger.WithField("error", err).Error("requestSync()")
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

func (n *Node) push(peer *peers.Peer, knownEvents map[uint32]int) error {
	//Check SyncLimit
	n.coreLock.Lock()
	overSyncLimit := n.core.OverSyncLimit(knownEvents, n.conf.SyncLimit, n.conf.EnableFastSync)
	n.coreLock.Unlock()

	if overSyncLimit {
		n.logger.Debug("SyncLimit")
		return nil
	}

	//Compute Diff
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
		//Convert to WireEvents
		wireEvents, err := n.core.ToWire(eventDiff)
		if err != nil {
			n.logger.WithField("error", err).Debug("Converting to WireEvent")
			return err
		}

		//Create and Send EagerSyncRequest
		start = time.Now()
		resp2, err := n.requestEagerSync(peer.NetAddr, wireEvents)
		elapsed = time.Since(start)
		n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestEagerSync()")
		if err != nil {
			n.logger.WithField("error", err).Error("requestEagerSync()")
			return err
		}
		n.logger.WithFields(logrus.Fields{
			"from_id": resp2.FromID,
			"success": resp2.Success,
		}).Debug("EagerSyncResponse")
	}

	return nil
}

func (n *Node) sync(fromID uint32, events []hg.WireEvent) error {
	//Insert Events in Hashgraph and create new Head if necessary
	start := time.Now()
	err := n.core.Sync(fromID, events)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Sync()")
	if err != nil {
		n.logger.WithError(err).Error()
		return err
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

func (n *Node) addTransaction(tx []byte) {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	n.core.AddTransactions([][]byte{tx})
}

//Shutdown shuts down the node
func (n *Node) Shutdown() {
	if n.getState() != Shutdown {
		n.logger.Debug("Shutdown")

		//Exit any non-shutdown state immediately
		n.setState(Shutdown)

		//Stop and wait for concurrent operations
		close(n.shutdownCh)

		n.waitRoutines()

		//For some reason this needs to be called after closing the shutdownCh
		//Not entirely sure why...
		n.controlTimer.Shutdown()

		//transport and store should only be closed once all concurrent operations
		//are finished otherwise they will panic trying to use close objects
		n.trans.Close()

		n.core.hg.Store.Close()
	}
}

//GetStats returns stats
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
		"sync_rate":              strconv.FormatFloat(n.SyncRate(), 'f', 2, 64),
		"events_per_second":      strconv.FormatFloat(consensusEventsPerSecond, 'f', 2, 64),
		"rounds_per_second":      strconv.FormatFloat(consensusRoundsPerSecond, 'f', 2, 64),
		"round_events":           strconv.Itoa(n.core.GetLastCommitedRoundEventsCount()),
		"id":                     fmt.Sprint(n.validator.ID()),
		"state":                  n.getState().String(),
		"moniker":                n.validator.Moniker,
	}
	return s
}

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

//SyncRate returns the Sync Rate
func (n *Node) SyncRate() float64 {
	var syncErrorRate float64

	if n.syncRequests != 0 {
		syncErrorRate = float64(n.syncErrors) / float64(n.syncRequests)
	}

	return 1 - syncErrorRate
}

//GetBlock returns a block
func (n *Node) GetBlock(blockIndex int) (*hg.Block, error) {
	return n.core.hg.Store.GetBlock(blockIndex)
}

//GetEvents returns a map of known events
func (n *Node) GetEvents() (map[uint32]int, error) {
	res := n.core.KnownEvents()

	return res, nil
}

//ID returns the validator ID
func (n *Node) ID() uint32 {
	return n.validator.ID()
}

//GetPeers returns the peers
func (n *Node) GetPeers() []*peers.Peer {
	return n.core.peers.Peers
}

//GetGenesisPeers returns the genesis peers
func (n *Node) GetGenesisPeers() []*peers.Peer {
	return n.core.genesisPeers.Peers
}
