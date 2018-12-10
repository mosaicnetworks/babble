package node

import (
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/mosaicnetworks/babble/src/crypto"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

type Core struct {
	id     uint32
	key    *ecdsa.PrivateKey
	pubKey []byte
	hexID  string
	hg     *hg.Hashgraph

	//XXX this needs major refactoring. Be careful with race conditions and
	//deadlocks
	peers        *peers.PeerSet //[PubKey] => id
	peerSelector PeerSelector
	selectorLock sync.Mutex

	//Hash and Index of this instance's head Event
	Head string
	Seq  int

	//Hashes of the Events that are not tied to the Head. This is managed by the
	//Sync method. If the gossip condition is false (there is nothing
	//interesting to record), items are added to heads; if the gossip condition
	//is true, items are removed from heads and used to record a new self-event.
	//This functionality allows to not grow the hashgraph for no reason.
	heads []string

	transactionPool         [][]byte
	internalTransactionPool []hg.InternalTransaction
	blockSignaturePool      []hg.BlockSignature

	proxyCommitCallback proxy.CommitCallback

	promises map[string]*JoinPromise

	logger *logrus.Entry
}

func NewCore(
	id uint32,
	key *ecdsa.PrivateKey,
	peers *peers.PeerSet,
	store hg.Store,
	proxyCommitCallback proxy.CommitCallback,
	logger *logrus.Logger) *Core {

	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	logEntry := logger.WithField("id", id)

	peerSelector := NewRandomPeerSelector(peers, id)

	core := &Core{
		id:                      id,
		key:                     key,
		proxyCommitCallback:     proxyCommitCallback,
		peers:                   peers,
		peerSelector:            peerSelector,
		transactionPool:         [][]byte{},
		internalTransactionPool: []hg.InternalTransaction{},
		blockSignaturePool:      []hg.BlockSignature{},
		promises:                make(map[string]*JoinPromise),
		heads:                   []string{},
		logger:                  logEntry,
		Head:                    "",
		Seq:                     -1,
	}

	core.hg = hg.NewHashgraph(store, core.Commit, logEntry)

	//XXX This will create roots and set PeerSet for round 0, which is not
	//necessarily correct; what if this is a node that only joins the cluster
	//on the go? Doesnt really matter because it's going to get Reset.
	core.hg.Init(peers)

	return core
}

func (c *Core) ID() uint32 {
	return c.id
}

func (c *Core) PubKey() []byte {
	if c.pubKey == nil {
		c.pubKey = crypto.FromECDSAPub(&c.key.PublicKey)
	}
	return c.pubKey
}

func (c *Core) HexID() string {
	if c.hexID == "" {
		pubKey := c.PubKey()
		c.hexID = fmt.Sprintf("0x%X", pubKey)
	}
	return c.hexID
}

func (c *Core) SetHeadAndSeq() error {
	var head string
	var seq int

	//Add self if not in Repertoire yet
	if _, ok := c.hg.Store.RepertoireByID()[c.ID()]; !ok {
		c.logger.Debug("Not in repertoire yet.")
		err := c.hg.Store.AddParticipant(peers.NewPeer(c.HexID(), ""))
		if err != nil {
			c.logger.WithError(err).Error("Error adding self to Store")
			return err
		}
	}

	last, isRoot, err := c.hg.Store.LastEventFrom(c.HexID())
	if err != nil {
		return err
	}

	if isRoot {
		root, err := c.hg.Store.GetRoot(c.HexID())
		if err != nil {
			return err
		}
		head = root.SelfParent.Hash
		seq = root.SelfParent.Index
	} else {
		lastEvent, err := c.GetEvent(last)
		if err != nil {
			return err
		}
		head = last
		seq = lastEvent.Index()
	}

	c.Head = head
	c.Seq = seq

	c.logger.WithFields(logrus.Fields{
		"core.Head": c.Head,
		"core.Seq":  c.Seq,
		"is_root":   isRoot,
	}).Debugf("SetHeadAndSeq")

	return nil
}

func (c *Core) Bootstrap() error {
	return c.hg.Bootstrap()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) SignAndInsertSelfEvent(event *hg.Event) error {
	if err := event.Sign(c.key); err != nil {
		return err
	}
	return c.InsertEvent(event, true)
}

func (c *Core) InsertEvent(event *hg.Event, setWireInfo bool) error {
	if err := c.hg.InsertEvent(event, setWireInfo); err != nil {
		return err
	}
	if event.Creator() == c.HexID() {
		c.Head = event.Hex()
		c.Seq = event.Index()
	}
	return nil
}

func (c *Core) KnownEvents() map[uint32]int {
	return c.hg.Store.KnownEvents()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) Commit(block *hg.Block) error {
	//Commit the Block to the App
	commitResponse, err := c.proxyCommitCallback(*block)

	c.logger.WithFields(logrus.Fields{
		"block":                          block.Index(),
		"state_hash":                     fmt.Sprintf("%X", commitResponse.StateHash),
		"accepted_internal_transactions": commitResponse.AcceptedInternalTransactions,
		"err": err,
	}).Debug("CommitBlock Response")

	//XXX Handle errors

	//Handle the response to set Block StateHash and process accepted
	//InternalTransactions which might update the PeerSet.
	if err == nil {
		block.Body.StateHash = commitResponse.StateHash

		sig, err := c.SignBlock(block)
		if err != nil {
			return err
		}

		c.AddBlockSignature(sig)

		err = c.ProcessAcceptedInternalTransactions(block.RoundReceived(), commitResponse.AcceptedInternalTransactions)
		if err != nil {
			return err
		}
	}

	return err
}

func (c *Core) SignBlock(block *hg.Block) (hg.BlockSignature, error) {
	sig, err := block.Sign(c.key)
	if err != nil {
		return hg.BlockSignature{}, err
	}
	if err := block.SetSignature(sig); err != nil {
		return hg.BlockSignature{}, err
	}
	return sig, c.hg.Store.SetBlock(block)
}

func (c *Core) ProcessAcceptedInternalTransactions(roundReceived int, txs []hg.InternalTransaction) error {
	peers := c.peers

	changed := false
	for _, tx := range txs {
		//update the PeerSet placholder
		switch tx.Type {
		case hg.PEER_ADD:
			c.logger.WithField("peer", tx.Peer).Debug("adding peer")
			peers = peers.WithNewPeer(&tx.Peer)
		case hg.PEER_REMOVE:
			c.logger.WithField("peer", tx.Peer).Debug("removing peer")
			peers = peers.WithRemovedPeer(&tx.Peer)
		default:
		}

		changed = true
	}

	//Why +4? It is what we call the RoundDecided; the round of the first
	//witness that can decide the fame of a SuperMajority of witnesses from
	//roundReceived, also accounting for Coin rounds. Cf whitepaper proofs.
	acceptedRound := roundReceived + 4

	if changed {
		err := c.hg.Store.SetPeerSet(acceptedRound, peers)
		if err != nil {
			return fmt.Errorf("Udpating Store PeerSet: %s", err)
		}

		c.peers = peers
		c.peerSelector = NewRandomPeerSelector(peers, c.id)
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

func (c *Core) OverSyncLimit(knownEvents map[uint32]int, syncLimit int) bool {
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

func (c *Core) GetAnchorBlockWithFrame() (*hg.Block, *hg.Frame, error) {
	return c.hg.GetAnchorBlockWithFrame()
}

//returns events that c knowns about and are not in 'known'
func (c *Core) EventDiff(known map[uint32]int) (events []*hg.Event, err error) {
	unknown := []*hg.Event{}
	//known represents the index of the last event known for every participant
	//compare this to our view of events and fill unknown with events that we know of
	// and the other doesnt
	for id, ct := range known {
		peer, ok := c.peers.ByID[id]

		if !ok {
			continue
		}

		//get participant Events with index > ct
		participantEvents, err := c.hg.Store.ParticipantEvents(peer.PubKeyHex, ct)
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
	c.logger.WithFields(logrus.Fields{
		"unknown_events":            len(unknownEvents),
		"transaction_pool":          len(c.transactionPool),
		"internal_transaction_pool": len(c.internalTransactionPool),
		"block_signature_pool":      len(c.blockSignaturePool),
	}).Debug("Sync")

	otherHead := ""
	for _, we := range unknownEvents {
		ev, err := c.hg.ReadWireInfo(we)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"wire_event": we,
				"error":      err,
			}).Error("Reading WireEvent")
			return err
		}

		if err := c.InsertEvent(ev, false); err != nil {
			c.logger.WithError(err).Errorf("Inserting Event %s", ev.Hex())
			return err
		}

		if we.Body.CreatorID == fromID {
			otherHead = ev.Hex()
		}
	}

	c.heads = append(c.heads, otherHead)

	//Create new event with self head and other head only if there are pending
	//loaded events or the pools are not empty
	if c.hg.PendingLoadedEvents > 0 ||
		len(c.transactionPool) > 0 ||
		len(c.internalTransactionPool) > 0 ||
		len(c.blockSignaturePool) > 0 {

		return c.RecordHeads()
	}

	return nil
}

func (c *Core) RecordHeads() error {
	handledHeads := 0
	defer func() {
		c.heads = c.heads[handledHeads:]
	}()

	for _, b := range c.heads {
		if err := c.AddSelfEvent(b); err != nil {
			return err
		}
		handledHeads++
	}

	return nil
}

func (c *Core) AddSelfEvent(otherHead string) error {
	//create new event with self head and otherHead
	//empty pools in its payload
	newHead := hg.NewEvent(c.transactionPool,
		c.internalTransactionPool,
		c.blockSignaturePool,
		[]string{c.Head, otherHead},
		c.PubKey(), c.Seq+1)

	if err := c.SignAndInsertSelfEvent(newHead); err != nil {
		return fmt.Errorf("Error inserting new head: %s", err)
	}

	c.logger.WithFields(logrus.Fields{
		"transactions":          len(c.transactionPool),
		"internal_transactions": len(c.internalTransactionPool),
		"block_signatures":      len(c.blockSignaturePool),
	}).Debug("Created Self-Event")

	c.transactionPool = [][]byte{}
	c.internalTransactionPool = []hg.InternalTransaction{}
	c.blockSignaturePool = []hg.BlockSignature{}

	return nil
}

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

	err = c.RunConsensus()
	if err != nil {
		return err
	}

	return nil
}

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

func (c *Core) ToWire(events []*hg.Event) ([]hg.WireEvent, error) {
	wireEvents := make([]hg.WireEvent, len(events), len(events))

	for i, e := range events {
		wireEvents[i] = e.ToWire()
	}

	return wireEvents, nil
}

func (c *Core) RunConsensus() error {
	start := time.Now()

	err := c.hg.DivideRounds()

	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("DivideRounds()")

	if err != nil {
		c.logger.WithField("error", err).Error("DivideRounds")

		return err
	}

	start = time.Now()

	err = c.hg.DecideFame()

	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("DecideFame()")

	if err != nil {
		c.logger.WithField("error", err).Error("DecideFame")

		return err
	}

	start = time.Now()

	err = c.hg.DecideRoundReceived()

	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("DecideRoundReceived()")

	if err != nil {
		c.logger.WithField("error", err).Error("DecideRoundReceived")

		return err
	}

	start = time.Now()

	err = c.hg.ProcessDecidedRounds()

	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("ProcessDecidedRounds()")

	if err != nil {
		c.logger.WithField("error", err).Error("ProcessDecidedRounds")

		return err
	}

	start = time.Now()

	err = c.hg.ProcessSigPool()

	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("ProcessSigPool()")

	if err != nil {
		c.logger.WithField("error", err).Error("ProcessSigPool()")

		return err
	}

	return nil
}

func (c *Core) AddTransactions(txs [][]byte) {
	c.transactionPool = append(c.transactionPool, txs...)
}

func (c *Core) AddInternalTransaction(tx hg.InternalTransaction) *JoinPromise {
	//create promise
	promise := NewJoinPromise(tx)

	//save it to promise store, for later use by the Commit callback
	c.promises[tx.Hash()] = promise

	//submit the internal tx, it will be processed asynchronously by the gossip
	//routines
	c.internalTransactionPool = append(c.internalTransactionPool, tx)

	//return the promise
	return promise
}

func (c *Core) AddBlockSignature(bs hg.BlockSignature) {
	c.blockSignaturePool = append(c.blockSignaturePool, bs)
}

func (c *Core) GetHead() (*hg.Event, error) {
	return c.hg.Store.GetEvent(c.Head)
}

func (c *Core) GetEvent(hash string) (*hg.Event, error) {
	return c.hg.Store.GetEvent(hash)
}

func (c *Core) GetEventTransactions(hash string) ([][]byte, error) {
	var txs [][]byte
	ex, err := c.GetEvent(hash)
	if err != nil {
		return txs, err
	}
	txs = ex.Transactions()
	return txs, nil
}

func (c *Core) GetConsensusEvents() []string {
	return c.hg.Store.ConsensusEvents()
}

func (c *Core) GetConsensusEventsCount() int {
	return c.hg.Store.ConsensusEventsCount()
}

func (c *Core) GetUndeterminedEvents() []string {
	return c.hg.UndeterminedEvents
}

func (c *Core) GetPendingLoadedEvents() int {
	return c.hg.PendingLoadedEvents
}

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

func (c *Core) GetLastConsensusRoundIndex() *int {
	return c.hg.LastConsensusRound
}

func (c *Core) GetConsensusTransactionsCount() int {
	return c.hg.ConsensusTransactions
}

func (c *Core) GetLastCommitedRoundEventsCount() int {
	return c.hg.LastCommitedRoundEvents
}

func (c *Core) GetLastBlockIndex() int {
	return c.hg.Store.LastBlockIndex()
}
