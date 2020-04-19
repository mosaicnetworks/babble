package node

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

// core is the object that is used by Node to manipulate the hashgraph
// indirectly.
type core struct {

	// validator is a wrapper around the private-key controlling this node.
	validator *Validator

	// hg is the underlying hashgraph where all the consensus computation and
	// data reside.
	hg *hg.Hashgraph

	// genesisPeers is the validator-set that the hashgraph was initialised
	// with.
	genesisPeers *peers.PeerSet

	// validators reflects the latest validator-set used in the hashgraph
	// consensus methods.
	validators *peers.PeerSet

	// peers is the list of peers that the node will try to gossip with; not
	// necessarily the current validator-set.
	peers *peers.PeerSet

	// peerSelector is the object that decides which peer to talk to next.
	peerSelector peerSelector
	selectorLock sync.Mutex

	// Hash and Index of this instance's head Event
	head string
	seq  int

	// acceptedRound is the first round at which the node's last join request
	// (InternalTransaction) takes effect. A node will not create SelfEvents
	// before reaching AcceptedRound. Default -1.
	acceptedRound int

	// removedRound is the round at which the node's last leave request takes
	// effect (if there is one). Default -1.
	removedRound int

	// targetRound is the minimum Consensus Round that the node needs to reach.
	// It is useful to set this value to a joining peer's accepted-round to
	// prevent them from having to wait.
	targetRound int

	// lastPeerChangeRound is updated whenever a join / leave request is
	// accepted
	lastPeerChangeRound int

	// Events that are not tied to this node's Head. This is managed by the Sync
	// method. If the gossip condition is false (there is nothing interesting to
	// record), items are added to heads; if the gossip condition is true, items
	// are removed from heads and used to record a new self-event. This
	// functionality allows to not grow the hashgraph continuously when there is
	// nothing to record.
	heads map[uint32]*hg.Event

	// The transaction pool contains transactions submitted from the app that
	// still haven't made it into the hashgraph.
	transactionPool [][]byte

	// internalTransactionPool is the same as transactionPool but for
	// InternalTransactions
	internalTransactionPool []hg.InternalTransaction

	// selfBlockSignatures is a pool of block-signatures, created by this node,
	// that still haven't made it into the hashgraph.
	selfBlockSignatures *hg.SigPool

	// proxyCommitCallback is called by the hashgraph when a block is committed
	proxyCommitCallback proxy.CommitCallback

	// maintenanceMode is passed through the constructor to indicate whether the
	// user of core is in maintenance mode. This is used here to disable leave
	// requests when a node is in maintenance mode
	maintenanceMode bool

	// promises keeps track of pending JoinRequests while the corresponding
	// InternalTransactions go through consensus asynchronously.
	promises map[string]*joinPromise

	logger *logrus.Entry
}

// newCore creates and initialises a new Core object.
func newCore(
	validator *Validator,
	peers *peers.PeerSet,
	genesisPeers *peers.PeerSet,
	store hg.Store,
	proxyCommitCallback proxy.CommitCallback,
	maintenanceMode bool,
	logger *logrus.Entry) *core {

	peerSelector := newRandomPeerSelector(peers, validator.ID())

	core := &core{
		validator:               validator,
		proxyCommitCallback:     proxyCommitCallback,
		genesisPeers:            genesisPeers,
		validators:              genesisPeers,
		peers:                   peers,
		peerSelector:            peerSelector,
		transactionPool:         [][]byte{},
		internalTransactionPool: []hg.InternalTransaction{},
		selfBlockSignatures:     hg.NewSigPool(),
		promises:                make(map[string]*joinPromise),
		heads:                   make(map[uint32]*hg.Event),
		logger:                  logger,
		head:                    "",
		seq:                     -1,
		acceptedRound:           -1,
		removedRound:            -1,
		targetRound:             -1,
		lastPeerChangeRound:     -1,
		maintenanceMode:         maintenanceMode,
	}

	core.hg = hg.NewHashgraph(store, core.commit, logger)

	core.hg.Init(genesisPeers)

	return core
}

// SetHeadAndSeq sets the Head and Seq of a Core object
func (c *core) setHeadAndSeq() error {
	head := ""
	seq := -1

	_, ok := c.hg.Store.RepertoireByID()[c.validator.ID()]

	if ok {
		last, err := c.hg.Store.LastEventFrom(c.validator.PublicKeyHex())
		if err != nil && !common.IsStore(err, common.Empty) {
			return err
		}

		if last != "" {
			lastEvent, err := c.getEvent(last)
			if err != nil {
				return err
			}

			head = last
			seq = lastEvent.Index()
		}
	} else {
		c.logger.Debug("Not in repertoire yet.")
	}

	c.head = head
	c.seq = seq

	c.logger.WithFields(logrus.Fields{
		"core.Head": c.head,
		"core.Seq":  c.seq,
	}).Debugf("SetHeadAndSeq")

	return nil
}

// bootstrap calls the Hashgraph Bootstrap
func (c *core) bootstrap() error {
	c.logger.Debug("Bootstrap")
	return c.hg.Bootstrap()
}

// setPeers sets the peers property and a New RandomPeerSelector
func (c *core) setPeers(ps *peers.PeerSet) {
	c.peers = ps
	c.peerSelector = newRandomPeerSelector(c.peers, c.validator.ID())
}

/*******************************************************************************
Busy
*******************************************************************************/

// busy indicates whether there is some unfinished work.
func (c *core) busy() bool {
	return c.hg.PendingLoadedEvents > 0 ||
		len(c.transactionPool) > 0 ||
		len(c.internalTransactionPool) > 0 ||
		c.selfBlockSignatures.Len() > 0 ||
		(c.hg.LastConsensusRound != nil && *c.hg.LastConsensusRound < c.targetRound)
}

/*******************************************************************************
Sync
*******************************************************************************/

// sync decodes and inserts new Events into the Hashgraph. UnknownEvents are
// expected to be in topoligical order.
func (c *core) sync(fromID uint32, unknownEvents []hg.WireEvent) error {
	c.logger.WithField("unknown_events", len(unknownEvents)).Debug("Sync")

	var otherHead *hg.Event
	for _, we := range unknownEvents {
		ev, err := c.hg.ReadWireInfo(we)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"wire_event": we,
				"error":      err,
			}).Error("Reading WireEvent")
			return err
		}

		// NormalSelfParentErrors are not reported. They can happen when two
		// concurrent pulls are trying to insert the same events.
		if err := c.insertEventAndRunConsensus(ev, false); err != nil {
			if hg.IsNormalSelfParentError(err) {
				continue
			} else {
				c.logger.WithError(err).Errorf("Inserting Event")
				return err
			}
		}

		if we.Body.CreatorID == fromID {
			otherHead = ev
		}

		if h, ok := c.heads[we.Body.CreatorID]; ok &&
			h != nil &&
			we.Body.Index > h.Index() {

			delete(c.heads, we.Body.CreatorID)
		}
	}

	// Do not overwrite a non-empty head with an empty head
	if h, ok := c.heads[fromID]; !ok ||
		h == nil ||
		(otherHead != nil && otherHead.Index() > h.Index()) {

		c.heads[fromID] = otherHead
	}

	c.logger.WithFields(logrus.Fields{
		"loaded_events":             c.hg.PendingLoadedEvents,
		"transaction_pool":          len(c.transactionPool),
		"internal_transaction_pool": len(c.internalTransactionPool),
		"self_signature_pool":       c.selfBlockSignatures.Len(),
		"target_round":              c.targetRound,
	}).Debug("Sync")

	// Create new event with self head and other head only if there are pending
	// loaded events or the pools are not empty
	if c.busy() ||
		c.seq < 0 {
		return c.recordHeads()
	}

	return nil
}

// recordHeads adds heads as SelfEvents
func (c *core) recordHeads() error {
	c.logger.WithField("heads", len(c.heads)).Debug("RecordHeads()")

	for id, ev := range c.heads {
		op := ""
		if ev != nil {
			op = ev.Hex()
		}
		if err := c.addSelfEvent(op); err != nil {
			return err
		}
		delete(c.heads, id)
	}

	return nil
}

// addSelfEvent adds a self event
func (c *core) addSelfEvent(otherHead string) error {
	if c.hg.Store.LastRound() < c.acceptedRound {
		c.logger.Debugf("Too early to insert self-event (%d / %d)", c.hg.Store.LastRound(), c.acceptedRound)
		return nil
	}

	// Add own block signatures to next Event
	sigs := c.selfBlockSignatures.Slice()
	txs := len(c.transactionPool)
	itxs := len(c.internalTransactionPool)

	// create new event with self head and otherHead, and empty pools in its
	// payload
	newHead := hg.NewEvent(c.transactionPool,
		c.internalTransactionPool,
		sigs,
		[]string{c.head, otherHead},
		c.validator.PublicKeyBytes(),
		c.seq+1)

	// Inserting the Event, and running consensus methods, can have a
	// side-effect of adding items to the transaction pools (via the commit
	// callback).
	if err := c.signAndInsertSelfEvent(newHead); err != nil {
		c.logger.WithError(err).Errorf("Error inserting new head")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"index":                 newHead.Index(),
		"transactions":          len(newHead.Transactions()),
		"internal_transactions": len(newHead.InternalTransactions()),
		"block_signatures":      len(newHead.BlockSignatures()),
	}).Debug("Created Self-Event")

	// do not remove pool elements that were added by CommitCallback
	c.transactionPool = c.transactionPool[txs:]
	c.internalTransactionPool = c.internalTransactionPool[itxs:]
	c.selfBlockSignatures.RemoveSlice(sigs)

	return nil
}

// signAndInsertSelfEvent signs a Hashgraph Event, inserts it and runs
// consensus.
func (c *core) signAndInsertSelfEvent(event *hg.Event) error {
	if err := event.Sign(c.validator.Key); err != nil {
		return err
	}
	return c.insertEventAndRunConsensus(event, true)
}

// insertEventAndRunConsensus Inserts a hashgraph event and runs consensus
func (c *core) insertEventAndRunConsensus(event *hg.Event, setWireInfo bool) error {
	if err := c.hg.InsertEventAndRunConsensus(event, setWireInfo); err != nil {
		return err
	}
	if event.Creator() == c.validator.PublicKeyHex() {
		c.head = event.Hex()
		c.seq = event.Index()
	}
	return nil
}

// knownEvents returns known events from the Hashgraph store
func (c *core) knownEvents() map[uint32]int {
	return c.hg.Store.KnownEvents()
}

/*******************************************************************************
FastForward
*******************************************************************************/

// fastForward is used whilst in CatchingUp state to reset the underlying
// hashgraph from a Block and associated Frame.
func (c *core) fastForward(block *hg.Block, frame *hg.Frame) error {
	c.logger.Debug("Fast Forward", frame.Round)
	peerSet := peers.NewPeerSet(frame.Peers)

	// Check Block Signatures
	err := c.hg.CheckBlock(block, peerSet)
	if err != nil {
		return err
	}

	// Check Frame Hash
	frameHash, err := frame.Hash()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(block.FrameHash(), frameHash) {
		return fmt.Errorf("Invalid Frame Hash")
	}

	err = c.hg.Reset(block, frame)
	if err != nil {
		return err
	}

	err = c.setHeadAndSeq()
	if err != nil {
		return err
	}

	// Update peer-selector and validators
	c.setPeers(peers.NewPeerSet(frame.Peers))
	c.validators = peers.NewPeerSet(frame.Peers)

	return nil
}

// getAnchorBlockWithFrame returns GetAnchorBlockWithFrame from the hashgraph
func (c *core) getAnchorBlockWithFrame() (*hg.Block, *hg.Frame, error) {
	return c.hg.GetAnchorBlockWithFrame()
}

/*******************************************************************************
Leave
*******************************************************************************/

// leave causes the node to politely leave the network. If the node is not
// alone, it submits an InternalTransaction to be removed from the
// validator-set. Otherwise it does nothing.
func (c *core) leave(leaveTimeout time.Duration) error {
	// Do nothing if we are not a validator.
	p, ok := c.validators.ByID[c.validator.ID()]
	if !ok {
		c.logger.Debugf("Leave: not a validator, do nothing")
		return nil
	}

	// Do nothing if we are the only validator.
	if len(c.validators.Peers) <= 1 {
		c.logger.Debugf("Leave: alone, do nothing")
		return nil
	}

	// Check for maintenance mode, if set no need for a leave request
	if c.maintenanceMode {
		c.logger.Debugf("Leave: maintenance mode, do nothing")
		return nil
	}

	// Otherwise, submit an InternalTransaction
	c.logger.Debugf("Leave: submit InternalTransaction")

	itx := hg.NewInternalTransaction(hg.PEER_REMOVE, *p)
	itx.Sign(c.validator.Key)

	promise := c.addInternalTransaction(itx)

	// Wait for the InternalTransaction to go through consensus
	timeout := time.After(leaveTimeout)
	select {
	case resp := <-promise.respCh:
		c.logger.WithFields(logrus.Fields{
			"leaving_round": resp.acceptedRound,
			"peers":         len(resp.peers),
		}).Debug("leave request processed")
	case <-timeout:
		err := fmt.Errorf("Timeout waiting for leave request to go through consensus")
		c.logger.WithError(err).Error()
		return err
	}

	// Wait for node to reach RemovedRound
	if c.peers.Len() >= 1 {
		timeout = time.After(leaveTimeout)
		for {
			select {
			case <-timeout:
				err := fmt.Errorf("Timeout waiting for leaving node to reach TargetRound")
				c.logger.WithError(err).Error()
				return err
			default:
				if c.hg.LastConsensusRound != nil && *c.hg.LastConsensusRound < c.removedRound {
					c.logger.Debugf("Waiting to reach RemovedRound: %d/%d", *c.hg.LastConsensusRound, c.removedRound)
					time.Sleep(100 * time.Millisecond)
				} else {
					return nil
				}
			}
		}
	}

	return nil
}

/*******************************************************************************
Commit
*******************************************************************************/

// commit the Block to the App using the proxyCommitCallback
func (c *core) commit(block *hg.Block) error {
	c.logger.WithFields(logrus.Fields{
		"block":        block.Index(),
		"txs":          len(block.Transactions()),
		"internal_txs": len(block.InternalTransactions()),
	}).Info("Commit")

	// Commit the Block to the App
	commitResponse, err := c.proxyCommitCallback(*block)
	if err != nil {
		c.logger.WithError(err).Error("Commit response")
	}

	c.logger.WithFields(logrus.Fields{
		"block":                 block.Index(),
		"internal_txs_receipts": len(commitResponse.InternalTransactionReceipts),
		"state_hash":            common.EncodeToString(commitResponse.StateHash),
	}).Info("Commit response")

	// Handle the response to set Block StateHash and process receipts which
	// might update the PeerSet.
	if err == nil {
		block.Body.StateHash = commitResponse.StateHash
		block.Body.InternalTransactionReceipts = commitResponse.InternalTransactionReceipts

		// Sign the block if we belong to its validator-set
		blockPeerSet, err := c.hg.Store.GetPeerSet(block.RoundReceived())
		if err != nil {
			return err
		}

		if _, ok := blockPeerSet.ByID[c.validator.ID()]; ok {
			sig, err := c.signBlock(block)
			if err != nil {
				return err
			}
			c.selfBlockSignatures.Add(sig)
		}

		err = c.hg.SetAnchorBlock(block)
		if err != nil {
			return err
		}

		err = c.processAcceptedInternalTransactions(block.RoundReceived(), commitResponse.InternalTransactionReceipts)
		if err != nil {
			return err
		}
	}

	return err
}

// signBlock signs the block and saves it.
func (c *core) signBlock(block *hg.Block) (hg.BlockSignature, error) {
	sig, err := block.Sign(c.validator.Key)
	if err != nil {
		return hg.BlockSignature{}, err
	}

	err = block.SetSignature(sig)
	if err != nil {
		return hg.BlockSignature{}, err
	}

	err = c.hg.Store.SetBlock(block)
	if err != nil {
		return sig, err
	}

	return sig, nil
}

// processAcceptedInternalTransactions processes a list of
// InternalTransactionReceipts from a block, updates the PeerSet for the
// corresponding round (round-received + 6), and responds to eventual promises.
func (c *core) processAcceptedInternalTransactions(roundReceived int, receipts []hg.InternalTransactionReceipt) error {
	currentPeers := c.peers
	validators := c.validators

	// Why +6? According to lemmas 5.15 and 5.17 of the original whitepaper, all
	// consistent hashgraphs will have decided the fame of round r witnesses by
	// round r+5 or before; so it is safe to set the new peer-set at round r+6.
	effectiveRound := roundReceived + 6

	changed := false
	for _, r := range receipts {
		txBody := r.InternalTransaction.Body

		if r.Accepted {
			c.logger.WithFields(logrus.Fields{
				"peer":           txBody.Peer,
				"round_received": roundReceived,
				"type":           txBody.Type.String(),
			}).Debug("Processing accepted InternalTransaction")

			switch txBody.Type {
			case hg.PEER_ADD:
				validators = validators.WithNewPeer(&txBody.Peer)
				currentPeers = currentPeers.WithNewPeer(&txBody.Peer)
			case hg.PEER_REMOVE:
				validators = validators.WithRemovedPeer(&txBody.Peer)
				currentPeers = currentPeers.WithRemovedPeer(&txBody.Peer)

				// Update RemovedRound if removing self
				if txBody.Peer.ID() == c.validator.ID() {
					c.logger.Debugf("Update RemovedRound from %d to %d", c.removedRound, effectiveRound)
					c.removedRound = effectiveRound
				}
			default:
				c.logger.Errorf("Unknown InternalTransactionType %s", txBody.Type)
				continue
			}

			changed = true
		} else {
			c.logger.WithField("peer", txBody.Peer).Debug("InternalTransaction not accepted")
		}
	}

	if changed {
		// Record the new validator-set in the underlying Hashgraph and in the
		// core's validators field
		c.lastPeerChangeRound = effectiveRound

		err := c.hg.Store.SetPeerSet(effectiveRound, validators)
		if err != nil {
			return fmt.Errorf("Updating Store PeerSet: %s", err)
		}

		c.validators = validators

		c.logger.WithFields(logrus.Fields{
			"effective_round": effectiveRound,
			"validators":      len(validators.Peers),
		}).Info("Validators changed")

		// Update the current list of communicating peers. This is not
		// necessarily equal to the latest recorded validator_set.
		c.setPeers(currentPeers)

		// A new validator-set has been recorded and will only be effective from
		// effectiveRound. A joining node will not be able to participate in the
		// consensus until the Hashgraph reaches that effectiveRound. Hence, we
		// force everyone to reach that round.
		if effectiveRound > c.targetRound {
			c.logger.Debugf("Update TargetRound from %d to %d", c.targetRound, effectiveRound)
			c.targetRound = effectiveRound
		}
	}

	for _, r := range receipts {
		//respond to the corresponding promise
		if p, ok := c.promises[r.InternalTransaction.HashString()]; ok {
			if r.Accepted {
				p.respond(true, effectiveRound, c.validators.Peers)
			} else {
				p.respond(false, 0, []*peers.Peer{})
			}
			delete(c.promises, r.InternalTransaction.HashString())
		}
	}

	return nil
}

/*******************************************************************************
Diff
*******************************************************************************/

// eventDiff returns Events that we are aware of, and that are not known by
// another. They are returned in topological order. The parameter otherKnown is
// a map containing the last Event index per participant, as seen by another
// peer. We compare this to our view of events and return the diff.
func (c *core) eventDiff(otherKnown map[uint32]int) (events []*hg.Event, err error) {
	// unknown is the container for the Events that will be returned by this
	// method.
	unknown := []*hg.Event{}

	myknown := c.knownEvents()

	// We loop through our known map first
	for id := range myknown {

		ct, ok := otherKnown[id]

		// If the other is not yet aware of this validator. It will need all
		// it's events (starting at index -1).
		if !ok {
			ct = -1
		}

		peer, ok := c.hg.Store.RepertoireByID()[id]
		if !ok {
			continue
		}

		// get participant Events with index > ct
		participantEvents, err := c.hg.Store.ParticipantEvents(peer.PubKeyString(), ct)
		if err != nil {
			return []*hg.Event{}, err
		}

		for _, e := range participantEvents {
			ev, err := c.hg.Store.GetEvent(e)
			if err != nil {
				return []*hg.Event{}, err
			}

			unknown = append(unknown, ev)
		}

	}

	sort.Sort(hg.ByTopologicalOrder(unknown))

	return unknown, nil
}

// fromWire takes Wire Events and returns Hashgraph Events
func (c *core) fromWire(wireEvents []hg.WireEvent) ([]hg.Event, error) {
	events := make([]hg.Event, len(wireEvents), len(wireEvents))

	for i, w := range wireEvents {
		ev, err := c.hg.ReadWireInfo(w)
		if err != nil {
			return nil, err
		}

		events[i] = *ev
	}

	return events, nil
}

// toWire takes Hashgraph Events and returns Wire Events
func (c *core) toWire(events []*hg.Event) ([]hg.WireEvent, error) {
	wireEvents := make([]hg.WireEvent, len(events), len(events))

	for i, e := range events {
		wireEvents[i] = e.ToWire()
	}

	return wireEvents, nil
}

/*******************************************************************************
Pools
*******************************************************************************/

// processSigPool calls Hashgraph ProcessSigPool
func (c *core) processSigPool() error {
	return c.hg.ProcessSigPool()
}

// addTransactions appends transactions to the transaction pool
func (c *core) addTransactions(txs [][]byte) {
	c.transactionPool = append(c.transactionPool, txs...)
}

// addInternalTransaction adds an InternalTransaction to the  pool, and creates
// a corresponding promise.
func (c *core) addInternalTransaction(tx hg.InternalTransaction) *joinPromise {
	promise := newJoinPromise(tx)

	// Save it to promise store, for later use by the Commit callback
	c.promises[tx.HashString()] = promise

	// Submit the internal tx to be processed asynchronously by the gossip
	// routines
	c.internalTransactionPool = append(c.internalTransactionPool, tx)

	return promise
}

/*******************************************************************************
Getters
*******************************************************************************/

// getHead returns the head from the hashgraph store
func (c *core) getHead() (*hg.Event, error) {
	return c.hg.Store.GetEvent(c.head)
}

// getEvent returns an event from the hashgrapg store
func (c *core) getEvent(hash string) (*hg.Event, error) {
	return c.hg.Store.GetEvent(hash)
}

// getEventTransactions returns the transactions for an event
func (c *core) getEventTransactions(hash string) ([][]byte, error) {
	var txs [][]byte
	ex, err := c.getEvent(hash)
	if err != nil {
		return txs, err
	}
	txs = ex.Transactions()
	return txs, nil
}

// getConsensusEvents returns consensus events from the hashgragh store
func (c *core) getConsensusEvents() []string {
	return c.hg.Store.ConsensusEvents()
}

// getConsensusEventsCount returns the count of consensus events from the
// hashgragh store
func (c *core) getConsensusEventsCount() int {
	return c.hg.Store.ConsensusEventsCount()
}

// getUndeterminedEvents returns undetermined events from the hashgraph
func (c *core) getUndeterminedEvents() []string {
	return c.hg.UndeterminedEvents
}

// getPendingLoadedEvents returns pendign loading events from the hashgraph
func (c *core) getPendingLoadedEvents() int {
	return c.hg.PendingLoadedEvents
}

// getConsensusTransactions returns the transaction from the events returned by
// getConsensusEvents()
func (c *core) getConsensusTransactions() ([][]byte, error) {
	txs := [][]byte{}
	for _, e := range c.getConsensusEvents() {
		eTxs, err := c.getEventTransactions(e)
		if err != nil {
			return txs, fmt.Errorf("Consensus event not found: %s", e)
		}
		txs = append(txs, eTxs...)
	}
	return txs, nil
}

// getLastConsensusRoundIndex returns the Last Consensus Round from the hashgraph
func (c *core) getLastConsensusRoundIndex() *int {
	return c.hg.LastConsensusRound
}

// getConsensusTransactionsCount return ConsensusTransacions from the hashgraph
func (c *core) getConsensusTransactionsCount() int {
	return c.hg.ConsensusTransactions
}

// getLastCommitedRoundEventsCount returns LastCommitedRoundEvents from the
// hashgraph
func (c *core) getLastCommitedRoundEventsCount() int {
	return c.hg.LastCommitedRoundEvents
}

// getLastBlockIndex returns last block index from the hashgraph store
func (c *core) getLastBlockIndex() int {
	return c.hg.Store.LastBlockIndex()
}
