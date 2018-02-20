package node

import (
	"crypto/ecdsa"
	"fmt"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/babbleio/babble/crypto"
	hg "github.com/babbleio/babble/hashgraph"
)

type Core struct {
	id     int
	key    *ecdsa.PrivateKey
	pubKey []byte
	hexID  string
	hg     *hg.Hashgraph

	participants        map[string]int //[PubKey] => id
	reverseParticipants map[int]string //[id] => PubKey
	Head                string
	Seq                 int

	transactionPool    [][]byte
	blockSignaturePool []hg.BlockSignature

	logger *logrus.Logger
}

func NewCore(
	id int,
	key *ecdsa.PrivateKey,
	participants map[string]int,
	store hg.Store,
	commitCh chan hg.Block,
	logger *logrus.Logger) Core {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	reverseParticipants := make(map[int]string)
	for pk, id := range participants {
		reverseParticipants[id] = pk
	}

	core := Core{
		id:                  id,
		key:                 key,
		hg:                  hg.NewHashgraph(participants, store, commitCh, logger),
		participants:        participants,
		reverseParticipants: reverseParticipants,
		transactionPool:     [][]byte{},
		blockSignaturePool:  []hg.BlockSignature{},
		logger:              logger,
	}
	return core
}

func (c *Core) ID() int {
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

func (c *Core) Init() error {
	//Create and save the first Event
	initialEvent := hg.NewEvent([][]byte(nil), nil,
		[]string{"", ""},
		c.PubKey(),
		c.Seq)
	//We want to make the initial Event deterministic so that when a node is
	//restarted it will initilaize the same Event. cf. issues 19 and 10
	initialEvent.Body.Timestamp = time.Time{}.UTC()
	err := c.SignAndInsertSelfEvent(initialEvent)
	c.logger.WithFields(logrus.Fields{
		"index": initialEvent.Index(),
		"hash":  initialEvent.Hex()}).Debug("Initial Event")
	return err
}

func (c *Core) Bootstrap() error {
	if err := c.hg.Bootstrap(); err != nil {
		return err
	}

	var head string
	var seq int

	last, isRoot, err := c.hg.Store.LastEventFrom(c.HexID())
	if err != nil {
		return err
	}

	if isRoot {
		root, err := c.hg.Store.GetRoot(c.HexID())
		if err != nil {
			head = root.X
			seq = root.Index
		}
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

	return nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) SignAndInsertSelfEvent(event hg.Event) error {
	if err := event.Sign(c.key); err != nil {
		return err
	}
	if err := c.InsertEvent(event, true); err != nil {
		return err
	}
	return nil
}

func (c *Core) InsertEvent(event hg.Event, setWireInfo bool) error {
	if err := c.hg.InsertEvent(event, setWireInfo); err != nil {
		return err
	}
	if event.Creator() == c.HexID() {
		c.Head = event.Hex()
		c.Seq = event.Index()
	}
	return nil
}

func (c *Core) KnownEvents() map[int]int {
	return c.hg.KnownEvents()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) SignBlock(block hg.Block) (hg.BlockSignature, error) {
	sig, err := block.Sign(c.key)
	if err != nil {
		return hg.BlockSignature{}, err
	}
	if err := block.SetSignature(sig); err != nil {
		return hg.BlockSignature{}, err
	}
	return sig, c.hg.Store.SetBlock(block)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) OverSyncLimit(knownEvents map[int]int, syncLimit int) bool {
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

func (c *Core) GetFrame() (hg.Frame, error) {
	return c.hg.GetFrame()
}

//returns events that c knowns about and are not in 'known'
func (c *Core) EventDiff(known map[int]int) (events []hg.Event, err error) {
	unknown := []hg.Event{}
	//known represents the indez of the last event known for every participant
	//compare this to our view of events and fill unknown with events that we know of
	// and the other doesnt
	for id, ct := range known {
		pk := c.reverseParticipants[id]
		//get participant Events with index > ct
		participantEvents, err := c.hg.Store.ParticipantEvents(pk, ct)
		if err != nil {
			return []hg.Event{}, err
		}
		for _, e := range participantEvents {
			ev, err := c.hg.Store.GetEvent(e)
			if err != nil {
				return []hg.Event{}, err
			}
			unknown = append(unknown, ev)
		}
	}
	sort.Sort(hg.ByTopologicalOrder(unknown))

	return unknown, nil
}

func (c *Core) Sync(unknownEvents []hg.WireEvent) error {

	c.logger.WithFields(logrus.Fields{
		"unknown_events":       len(unknownEvents),
		"transaction_pool":     len(c.transactionPool),
		"block_signature_pool": len(c.blockSignaturePool),
	}).Debug("Sync")

	otherHead := ""
	//add unknown events
	for k, we := range unknownEvents {
		ev, err := c.hg.ReadWireInfo(we)
		if err != nil {
			return err
		}
		if err := c.InsertEvent(*ev, false); err != nil {
			return err
		}
		//assume last event corresponds to other-head
		if k == len(unknownEvents)-1 {
			otherHead = ev.Hex()
		}
	}

	//create new event with self head and other head
	//only if there are pending loaded events or the pools are not empty
	if len(unknownEvents) > 0 ||
		len(c.transactionPool) > 0 ||
		len(c.blockSignaturePool) > 0 {

		newHead := hg.NewEvent(c.transactionPool, c.blockSignaturePool,
			[]string{c.Head, otherHead},
			c.PubKey(),
			c.Seq+1)

		if err := c.SignAndInsertSelfEvent(newHead); err != nil {
			return fmt.Errorf("Error inserting new head: %s", err)
		}

		//empty the pools
		c.transactionPool = [][]byte{}
		c.blockSignaturePool = []hg.BlockSignature{}
	}

	return nil
}

func (c *Core) AddSelfEvent() error {
	if len(c.transactionPool) == 0 && len(c.blockSignaturePool) == 0 {
		c.logger.Debug("Empty transaction pool and block signature pool")
		return nil
	}

	//create new event with self head and empty other parent
	//empty transaction pool in its payload
	newHead := hg.NewEvent(c.transactionPool,
		c.blockSignaturePool,
		[]string{c.Head, ""},
		c.PubKey(), c.Seq+1)

	if err := c.SignAndInsertSelfEvent(newHead); err != nil {
		return fmt.Errorf("Error inserting new head: %s", err)
	}

	c.logger.WithFields(logrus.Fields{
		"transactions":     len(c.transactionPool),
		"block_signatures": len(c.blockSignaturePool),
	}).Debug("Created Self-Event")

	c.transactionPool = [][]byte{}
	c.blockSignaturePool = []hg.BlockSignature{}

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

func (c *Core) ToWire(events []hg.Event) ([]hg.WireEvent, error) {
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
	err = c.hg.FindOrder()
	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("FindOrder()")
	if err != nil {
		c.logger.WithField("error", err).Error("FindOrder")
		return err
	}

	return nil
}

func (c *Core) AddTransactions(txs [][]byte) {
	c.transactionPool = append(c.transactionPool, txs...)
}

func (c *Core) AddBlockSignature(bs hg.BlockSignature) {
	c.blockSignaturePool = append(c.blockSignaturePool, bs)
}

func (c *Core) GetHead() (hg.Event, error) {
	return c.hg.Store.GetEvent(c.Head)
}

func (c *Core) GetEvent(hash string) (hg.Event, error) {
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
	return c.hg.ConsensusEvents()
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
	return c.hg.LastBlockIndex
}

func (c *Core) NeedGossip() bool {
	return c.hg.PendingLoadedEvents > 0 ||
		len(c.transactionPool) > 0 ||
		len(c.blockSignaturePool) > 0
}
