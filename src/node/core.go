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

//Core is the core Node object
type Core struct {
	validator *Validator

	hg *hg.Hashgraph

	peers        *peers.PeerSet //[PubKey] => id
	peerSelector PeerSelector
	selectorLock sync.Mutex

	//Hash and Index of this instance's head Event
	Head string
	Seq  int

	//AcceptedRound is the first Round to which this peer belongs. A node will
	//not create SelfEvents before reaching AcceptedRound.
	AcceptedRound int

	/*
		TargetRound is the minimum Consensus Round that the node needs to reach.
		It is useful to set this value to a joining peer's accepted-round to
		prevent them from having to wait.
	*/
	TargetRound int

	/*
		Events that are not tied to this node's Head. This is managed by
		the Sync method. If the gossip condition is false (there is nothing
		interesting to record), items are added to heads; if the gossip
		condition is true, items are removed from heads and used to record a new
		self-event. This functionality allows to not grow the hashgraph
		continuously when there is nothing to record.
	*/
	heads map[uint32]*hg.Event

	transactionPool         [][]byte
	internalTransactionPool []hg.InternalTransaction
	selfBlockSignatures     *hg.SigPool

	proxyCommitCallback proxy.CommitCallback

	promises map[string]*JoinPromise

	logger *logrus.Entry
}

//NewCore returns a new Core object
func NewCore(
	validator *Validator,
	peers *peers.PeerSet,
	store hg.Store,
	proxyCommitCallback proxy.CommitCallback,
	logger *logrus.Logger) *Core {

	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	logEntry := logger.WithField("id", validator.ID())

	peerSelector := NewRandomPeerSelector(peers, validator.ID())

	core := &Core{
		validator:               validator,
		proxyCommitCallback:     proxyCommitCallback,
		peers:                   peers,
		peerSelector:            peerSelector,
		transactionPool:         [][]byte{},
		internalTransactionPool: []hg.InternalTransaction{},
		selfBlockSignatures:     hg.NewSigPool(),
		promises:                make(map[string]*JoinPromise),
		heads:                   make(map[uint32]*hg.Event),
		logger:                  logEntry,
		Head:                    "",
		Seq:                     -1,
		AcceptedRound:           -1,
		TargetRound:             -1,
	}

	core.hg = hg.NewHashgraph(store, core.Commit, logEntry)

	core.hg.Init(peers)

	return core
}

//SetHeadAndSeq sets the Head and Seq of a Core object
func (c *Core) SetHeadAndSeq() error {
	head := ""
	seq := -1

	_, ok := c.hg.Store.RepertoireByID()[c.validator.ID()]

	if ok {
		last, err := c.hg.Store.LastEventFrom(c.validator.PublicKeyHex())
		if err != nil && !common.Is(err, common.Empty) {
			return err
		}

		if last != "" {
			lastEvent, err := c.GetEvent(last)
			if err != nil {
				return err
			}

			head = last
			seq = lastEvent.Index()
		}
	} else {
		c.logger.Debug("Not in repertoire yet.")
	}

	c.Head = head
	c.Seq = seq

	c.logger.WithFields(logrus.Fields{
		"core.Head": c.Head,
		"core.Seq":  c.Seq,
	}).Debugf("SetHeadAndSeq")

	return nil
}

//Bootstrap calls the Hashgraph Bootstrap
func (c *Core) Bootstrap() error {
	return c.hg.Bootstrap()
}

//SetPeerSet sets the peers property and a New RandomPeerSelector
func (c *Core) SetPeerSet(ps *peers.PeerSet) {
	c.peers = ps
	c.peerSelector = NewRandomPeerSelector(c.peers, c.validator.ID())
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

//SignAndInsertSelfEvent signs a HashGraph Event, inserts it and runs consensus
func (c *Core) SignAndInsertSelfEvent(event *hg.Event) error {
	if err := event.Sign(c.validator.Key); err != nil {
		return err
	}
	return c.InsertEventAndRunConsensus(event, true)
}

//InsertEventAndRunConsensus Inserts a hashgraph event and runs consensus
func (c *Core) InsertEventAndRunConsensus(event *hg.Event, setWireInfo bool) error {
	if err := c.hg.InsertEventAndRunConsensus(event, setWireInfo); err != nil {
		return err
	}
	if event.Creator() == c.validator.PublicKeyHex() {
		c.Head = event.Hex()
		c.Seq = event.Index()
	}
	return nil
}

//KnownEvents returns known events from the Hashgraph store
func (c *Core) KnownEvents() map[uint32]int {
	return c.hg.Store.KnownEvents()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

//Commit the Block to the App
func (c *Core) Commit(block *hg.Block) error {
	//Commit the Block to the App
	commitResponse, err := c.proxyCommitCallback(*block)

	c.logger.WithFields(logrus.Fields{
		"block":                 block.Index(),
		"state_hash":            fmt.Sprintf("%X", commitResponse.StateHash),
		"internal_transactions": commitResponse.InternalTransactions,
		"err": err,
	}).Debug("CommitBlock Response")

	//XXX Handle errors

	//Handle the response to set Block StateHash and process accepted
	//InternalTransactions which might update the PeerSet.
	if err == nil {
		block.Body.StateHash = commitResponse.StateHash
		block.Body.InternalTransactions = commitResponse.InternalTransactions

		sig, err := c.SignBlock(block)
		if err != nil {
			return err
		}

		err = c.hg.SetAnchorBlock(block)
		if err != nil {
			return err
		}

		c.selfBlockSignatures.Add(sig)

		err = c.ProcessAcceptedInternalTransactions(block.RoundReceived(), commitResponse.InternalTransactions)
		if err != nil {
			return err
		}
	}

	return err
}

//SignBlock signs the block
func (c *Core) SignBlock(block *hg.Block) (hg.BlockSignature, error) {
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

//ProcessAcceptedInternalTransactions processes the accepted internal transactions
func (c *Core) ProcessAcceptedInternalTransactions(roundReceived int, txs []hg.InternalTransaction) error {
	peers := c.peers

	changed := false
	for _, tx := range txs {
		if tx.Body.Accepted == common.True {
			//update the PeerSet placeholder
			switch tx.Body.Type {
			case hg.PEER_ADD:
				c.logger.WithField("peer", tx.Body.Peer).Debug("adding peer")
				peers = peers.WithNewPeer(&tx.Body.Peer)
			case hg.PEER_REMOVE:
				c.logger.WithField("peer", tx.Body.Peer).Debug("removing peer")
				peers = peers.WithRemovedPeer(&tx.Body.Peer)
			default:
			}
			changed = true
		} else {
			c.logger.WithField("peer", tx.Body.Peer).Debugf("InternalTransaction not accepted. Got %v", tx.Body.Accepted)
		}

	}

	//Why +6? According to lemmas 5.15 and 5.17 of the original whitepaper, all
	//consistent hashgraphs will have decided the fame of round r witnesses by
	//round r+5 or before; so it is safe to set the new peer-set at round r+6.
	acceptedRound := roundReceived + 6

	if changed {
		err := c.hg.Store.SetPeerSet(acceptedRound, peers)
		if err != nil {
			return fmt.Errorf("Udpating Store PeerSet: %s", err)
		}

		//XXX should not be set immediately. We need a smarter way for core to
		//know which peerset to use depending on which round the hg is at.
		c.SetPeerSet(peers)

		//A new peer has joined and it won't be able to participate in consensus
		//until it fast-forwards to its accepted-round. Hence, we force the
		//other nodes to reach that round.
		if acceptedRound > c.TargetRound {
			c.TargetRound = acceptedRound
		}
	}

	for _, tx := range txs {
		//respond to the corresponding promise
		if p, ok := c.promises[tx.Hash()]; ok {
			p.Respond(acceptedRound, peers.Peers)
			delete(c.promises, tx.Hash())
		}
	}

	return nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

//OverSyncLimit checks if the number of unknown events exceeds the syncLimit
func (c *Core) OverSyncLimit(knownEvents map[uint32]int, syncLimit int, enableSyncLimit bool) bool {
	if !enableSyncLimit {
		return false
	}

	totUnknown := 0
	myKnownEvents := c.KnownEvents()
	for i, li := range myKnownEvents {
		if li > knownEvents[i] {
			totUnknown += li - knownEvents[i]
		}
	}
	if totUnknown > syncLimit {
		return true
	}
	return false
}

//GetAnchorBlockWithFrame returns GetAnchorBlockWithFrame from the hashgraph
func (c *Core) GetAnchorBlockWithFrame() (*hg.Block, *hg.Frame, error) {
	return c.hg.GetAnchorBlockWithFrame()
}

//EventDiff returns events that c knowns about and are not in 'known'
func (c *Core) EventDiff(known map[uint32]int) (events []*hg.Event, err error) {
	unknown := []*hg.Event{}
	//known represents the index of the last event known for every participant
	//compare this to our view of events and fill unknown with events that we know of
	// and the other doesnt
	for id, ct := range known {
		peer, ok := c.hg.Store.RepertoireByID()[id]
		if !ok {
			continue
		}

		//get participant Events with index > ct
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

//Sync decodes and inserts new Events into the Hashgraph. UnknownEvents are
//expected to be in topoligical order.
func (c *Core) Sync(fromID uint32, unknownEvents []hg.WireEvent) error {
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

		if err := c.InsertEventAndRunConsensus(ev, false); err != nil {
			c.logger.WithError(err).Errorf("Inserting Event")
			return err
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

	//Do not overwrite a non-empty head with an empty head
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
		"target_round":              c.TargetRound,
	}).Debug("Sync")

	//Create new event with self head and other head only if there are pending
	//loaded events or the pools are not empty
	if c.Busy() {
		return c.RecordHeads()
	}

	return nil
}

//RecordHeads adds heads as SelfEvents
func (c *Core) RecordHeads() error {
	c.logger.WithField("heads", len(c.heads)).Debug("RecordHeads()")

	for id, ev := range c.heads {
		op := ""
		if ev != nil {
			op = ev.Hex()
		}
		if err := c.AddSelfEvent(op); err != nil {
			return err
		}
		delete(c.heads, id)
	}

	return nil
}

//AddSelfEvent adds a self event
func (c *Core) AddSelfEvent(otherHead string) error {
	if c.hg.Store.LastRound() < c.AcceptedRound {
		c.logger.Debugf("Too early to insert self-event (%d / %d)", c.hg.Store.LastRound(), c.AcceptedRound)
		return nil
	}

	//Add own block signatures to next Event
	sigs := c.selfBlockSignatures.Slice()
	txs := len(c.transactionPool)
	itxs := len(c.internalTransactionPool)

	//create new event with self head and otherHead, and empty pools in its
	//payload
	newHead := hg.NewEvent(c.transactionPool,
		c.internalTransactionPool,
		sigs,
		[]string{c.Head, otherHead},
		c.validator.PublicKeyBytes(),
		c.Seq+1)

	//Inserting the Event, and running consensus methods, can have a side-effect
	//of adding items to the transaction pools (via the commit callback).
	if err := c.SignAndInsertSelfEvent(newHead); err != nil {
		c.logger.WithError(err).Errorf("Error inserting new head")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"index":                 newHead.Index(),
		"transactions":          len(newHead.Transactions()),
		"internal_transactions": len(newHead.InternalTransactions()),
		"block_signatures":      len(newHead.BlockSignatures()),
	}).Debug("Created Self-Event")

	//do not remove pool elements that were added by CommitCallback
	c.transactionPool = c.transactionPool[txs:]
	c.internalTransactionPool = c.internalTransactionPool[itxs:]
	c.selfBlockSignatures.RemoveSlice(sigs)

	return nil
}

//FastForward is used whilst in catchingUp state to apply past blocks and frames
func (c *Core) FastForward(peer string, block *hg.Block, frame *hg.Frame) error {
	peerSet := peers.NewPeerSet(frame.Peers)

	//Check Block Signatures
	err := c.hg.CheckBlock(block, peerSet)
	if err != nil {
		return err
	}

	//Check Frame Hash
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

	err = c.SetHeadAndSeq()
	if err != nil {
		return err
	}

	c.SetPeerSet(peers.NewPeerSet(frame.Peers))

	return nil
}

//Leave causes the node to leave the network
func (c *Core) Leave(leaveTimeout time.Duration) error {
	p, ok := c.peers.ByID[c.validator.ID()]
	if !ok {
		return fmt.Errorf("Leaving: Peer not found")
	}

	itx := hg.NewInternalTransaction(hg.PEER_REMOVE, *p)
	itx.Sign(c.validator.Key)

	promise := c.AddInternalTransaction(itx)

	//Wait for the InternalTransaction to go through consensus
	timeout := time.After(leaveTimeout)
	select {
	case resp := <-promise.RespCh:
		c.logger.WithFields(logrus.Fields{
			"leaving_round": resp.AcceptedRound,
			"peers":         len(resp.Peers),
		}).Debug("LeaveRequest processed")
	case <-timeout:
		err := fmt.Errorf("Timeout waiting for LeaveRequest to go through consensus")
		c.logger.WithError(err).Error()
		return err
	}

	if c.peers.Len() >= 1 {
		//Wait for node to reach accepted round
		timeout = time.After(leaveTimeout)
		for {
			select {
			case <-timeout:
				err := fmt.Errorf("Timeout waiting for leaving node to reach TargetRound")
				c.logger.WithError(err).Error()
				return err
			default:
				if c.hg.LastConsensusRound != nil && *c.hg.LastConsensusRound < c.TargetRound {
					c.logger.Debugf("Waiting to reach TargetRound: %d/%d", *c.hg.LastConsensusRound, c.TargetRound)
					time.Sleep(100 * time.Millisecond)
				} else {
					return nil
				}
			}
		}
	}

	return nil
}

//FromWire takes Wire Events and returns HashGraph Events
func (c *Core) FromWire(wireEvents []hg.WireEvent) ([]hg.Event, error) {
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

//ToWire takes HashGraph Events and returns Wire Events
func (c *Core) ToWire(events []*hg.Event) ([]hg.WireEvent, error) {
	wireEvents := make([]hg.WireEvent, len(events), len(events))

	for i, e := range events {
		wireEvents[i] = e.ToWire()
	}

	return wireEvents, nil
}

//ProcessSigPool calls Hashgraph ProcessSigPool
func (c *Core) ProcessSigPool() error {
	return c.hg.ProcessSigPool()
}

//AddTransactions appends transactions to the transaction pool
func (c *Core) AddTransactions(txs [][]byte) {
	c.transactionPool = append(c.transactionPool, txs...)
}

//AddInternalTransaction adds an internal transaction
func (c *Core) AddInternalTransaction(tx hg.InternalTransaction) *JoinPromise {
	//create promise
	promise := NewJoinPromise(tx)

	//save it to promise store, for later use by the Commit callback
	c.promises[tx.Hash()] = promise

	//submit the internal tx to be processed asynchronously by the gossip
	//routines
	c.internalTransactionPool = append(c.internalTransactionPool, tx)

	//return the promise
	return promise
}

//GetHead returns the head from the hashgraph store
func (c *Core) GetHead() (*hg.Event, error) {
	return c.hg.Store.GetEvent(c.Head)
}

//GetEvent returns an event from the hashgrapg store
func (c *Core) GetEvent(hash string) (*hg.Event, error) {
	return c.hg.Store.GetEvent(hash)
}

//GetEventTransactions returns the transactions for an event
func (c *Core) GetEventTransactions(hash string) ([][]byte, error) {
	var txs [][]byte
	ex, err := c.GetEvent(hash)
	if err != nil {
		return txs, err
	}
	txs = ex.Transactions()
	return txs, nil
}

//GetConsensusEvents returns consensus events from the hashgragh store
func (c *Core) GetConsensusEvents() []string {
	return c.hg.Store.ConsensusEvents()
}

//GetConsensusEventsCount returns the count of consensus events from the hashgragh store
func (c *Core) GetConsensusEventsCount() int {
	return c.hg.Store.ConsensusEventsCount()
}

//GetUndeterminedEvents returns undetermined events from the hashgraph
func (c *Core) GetUndeterminedEvents() []string {
	return c.hg.UndeterminedEvents
}

//GetPendingLoadedEvents returns pendign loading events from the hashgraph
func (c *Core) GetPendingLoadedEvents() int {
	return c.hg.PendingLoadedEvents
}

//GetConsensusTransactions returns the transaction from the events returned by GetConsensusEvents()
func (c *Core) GetConsensusTransactions() ([][]byte, error) {
	txs := [][]byte{}
	for _, e := range c.GetConsensusEvents() {
		eTxs, err := c.GetEventTransactions(e)
		if err != nil {
			return txs, fmt.Errorf("Consensus event not found: %s", e)
		}
		txs = append(txs, eTxs...)
	}
	return txs, nil
}

//GetLastConsensusRoundIndex returns the Last Consensus Round from the hashgraph
func (c *Core) GetLastConsensusRoundIndex() *int {
	return c.hg.LastConsensusRound
}

//GetConsensusTransactionsCount return ConsensusTransacions from the hashgraph
func (c *Core) GetConsensusTransactionsCount() int {
	return c.hg.ConsensusTransactions
}

//GetLastCommitedRoundEventsCount returns LastCommitedRoundEvents from the hashgraph
func (c *Core) GetLastCommitedRoundEventsCount() int {
	return c.hg.LastCommitedRoundEvents
}

//GetLastBlockIndex returns last block index from the hashgraph store
func (c *Core) GetLastBlockIndex() int {
	return c.hg.Store.LastBlockIndex()
}

//Busy returns a boolean that denotes whether there is incomplete processing
func (c *Core) Busy() bool {
	return c.hg.PendingLoadedEvents > 0 ||
		len(c.transactionPool) > 0 ||
		len(c.internalTransactionPool) > 0 ||
		c.selfBlockSignatures.Len() > 0 ||
		(c.hg.LastConsensusRound != nil && *c.hg.LastConsensusRound < c.TargetRound)
}
