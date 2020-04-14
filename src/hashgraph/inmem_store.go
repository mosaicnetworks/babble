package hashgraph

import (
	"strconv"

	cm "github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
)

// InmemStore implements the Store interface with inmemory caches. When the
// caches are full, older items are evicted, so InmemStore is not suitable for
// long running deployments where joining nodes expect to sync from the
// beginning of a hashgraph.
type InmemStore struct {
	cacheSize              int
	eventCache             *cm.LRU          //hash => Event
	roundCache             *cm.LRU          //round number => Round
	blockCache             *cm.LRU          //index => Block
	frameCache             *cm.LRU          //round received => Frame
	consensusCache         *cm.RollingIndex //consensus index => hash
	totConsensusEvents     int
	peerSetCache           *PeerSetCache           //start round => PeerSet
	participantEventsCache *ParticipantEventsCache //pubkey => Events
	roots                  map[string]*Root        //[participant] => Root
	lastRound              int
	lastConsensusEvents    map[string]string //[participant] => hex() of last consensus event
	lastBlock              int
}

// NewInmemStore creates a new InmemStore where all caches are limited by
// cacheSize items.
func NewInmemStore(cacheSize int) *InmemStore {
	store := &InmemStore{
		cacheSize:              cacheSize,
		eventCache:             cm.NewLRU(cacheSize, nil),
		roundCache:             cm.NewLRU(cacheSize, nil),
		blockCache:             cm.NewLRU(cacheSize, nil),
		frameCache:             cm.NewLRU(cacheSize, nil),
		consensusCache:         cm.NewRollingIndex("ConsensusCache", cacheSize),
		peerSetCache:           NewPeerSetCache(),
		participantEventsCache: NewParticipantEventsCache(cacheSize),
		roots:                  make(map[string]*Root),
		lastRound:              -1,
		lastBlock:              -1,
		lastConsensusEvents:    map[string]string{},
	}
	return store
}

// CacheSize returns the size limit that was provided to all the caches that
// make up the InmemStore. This does not correspond to the total number of items
// in the store or the total number of items allowed in the store.
func (s *InmemStore) CacheSize() int {
	return s.cacheSize
}

// GetPeerSet implements the Store interface.
func (s *InmemStore) GetPeerSet(round int) (*peers.PeerSet, error) {
	return s.peerSetCache.Get(round)
}

// SetPeerSet implements the Store interface.
func (s *InmemStore) SetPeerSet(round int, peerSet *peers.PeerSet) error {
	//Update PeerSetCache
	err := s.peerSetCache.Set(round, peerSet)
	if err != nil {
		return err
	}

	for _, p := range peerSet.Peers {
		s.addParticipant(p)
	}

	return nil
}

func (s *InmemStore) addParticipant(p *peers.Peer) error {
	if _, ok := s.participantEventsCache.participants.ByID[p.ID()]; !ok {
		if err := s.participantEventsCache.AddPeer(p); err != nil {
			return err
		}
	}

	if _, ok := s.roots[p.PubKeyString()]; !ok {
		s.roots[p.PubKeyString()] = NewRoot()
	}

	return nil
}

// GetAllPeerSets implements the Store interface.
func (s *InmemStore) GetAllPeerSets() (map[int][]*peers.Peer, error) {
	return s.peerSetCache.GetAll()
}

// FirstRound implements the Store interface.
func (s *InmemStore) FirstRound(id uint32) (int, bool) {
	return s.peerSetCache.FirstRound(id)
}

// RepertoireByPubKey implements the Store interface.
func (s *InmemStore) RepertoireByPubKey() map[string]*peers.Peer {
	return s.peerSetCache.RepertoireByPubKey()
}

// RepertoireByID implements the Store interface.
func (s *InmemStore) RepertoireByID() map[uint32]*peers.Peer {
	return s.peerSetCache.RepertoireByID()
}

// GetEvent implements the Store interface.
func (s *InmemStore) GetEvent(key string) (*Event, error) {
	res, ok := s.eventCache.Get(key)
	if !ok {
		return nil, cm.NewStoreErr("EventCache", cm.KeyNotFound, key)
	}

	return res.(*Event), nil
}

// SetEvent implements the Store interface.
func (s *InmemStore) SetEvent(event *Event) error {
	key := event.Hex()
	_, err := s.GetEvent(key)
	if err != nil && !cm.IsStore(err, cm.KeyNotFound) {
		return err
	}
	if cm.IsStore(err, cm.KeyNotFound) {
		if err := s.participantEventsCache.Set(event.Creator(), key, event.Index()); err != nil {
			return err
		}
	}
	s.eventCache.Add(key, event)
	return nil
}

// ParticipantEvents implements the Store interface.
func (s *InmemStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	return s.participantEventsCache.Get(participant, skip)
}

// ParticipantEvent implements the Store interface.
func (s *InmemStore) ParticipantEvent(participant string, index int) (string, error) {
	return s.participantEventsCache.GetItem(participant, index)
}

// LastEventFrom implements the Store interface.
func (s *InmemStore) LastEventFrom(participant string) (last string, err error) {
	last, err = s.participantEventsCache.GetLast(participant)
	return
}

// LastConsensusEventFrom implements the Store interface.
func (s *InmemStore) LastConsensusEventFrom(participant string) (last string, err error) {
	last, _ = s.lastConsensusEvents[participant]
	return
}

// KnownEvents implements the Store interface.
func (s *InmemStore) KnownEvents() map[uint32]int {
	return s.participantEventsCache.Known()
}

// ConsensusEvents implements the Store interface.
func (s *InmemStore) ConsensusEvents() []string {
	lastWindow, _ := s.consensusCache.GetLastWindow()
	res := make([]string, len(lastWindow))
	for i, item := range lastWindow {
		res[i] = item.(string)
	}
	return res
}

// ConsensusEventsCount implements the Store interface.
func (s *InmemStore) ConsensusEventsCount() int {
	return s.totConsensusEvents
}

// AddConsensusEvent implements the Store interface.
func (s *InmemStore) AddConsensusEvent(event *Event) error {
	s.consensusCache.Set(event.Hex(), s.totConsensusEvents)
	s.totConsensusEvents++
	s.lastConsensusEvents[event.Creator()] = event.Hex()
	return nil
}

// GetRound implements the Store interface.
func (s *InmemStore) GetRound(r int) (*RoundInfo, error) {
	res, ok := s.roundCache.Get(r)
	if !ok {
		return nil, cm.NewStoreErr("RoundCache", cm.KeyNotFound, strconv.Itoa(r))
	}
	return res.(*RoundInfo), nil
}

// SetRound implements the Store interface.
func (s *InmemStore) SetRound(r int, round *RoundInfo) error {
	s.roundCache.Add(r, round)
	if r > s.lastRound {
		s.lastRound = r
	}
	return nil
}

// LastRound implements the Store interface.
func (s *InmemStore) LastRound() int {
	return s.lastRound
}

// RoundWitnesses implements the Store interface.
func (s *InmemStore) RoundWitnesses(r int) []string {
	round, err := s.GetRound(r)
	if err != nil {
		return []string{}
	}
	return round.Witnesses()
}

// RoundEvents implements the Store interface.
func (s *InmemStore) RoundEvents(r int) int {
	round, err := s.GetRound(r)
	if err != nil {
		return 0
	}
	return len(round.CreatedEvents)
}

// GetRoot implements the Store interface.
func (s *InmemStore) GetRoot(participant string) (*Root, error) {
	res, ok := s.roots[participant]
	if !ok {
		return nil, cm.NewStoreErr("RootCache", cm.KeyNotFound, participant)
	}
	return res, nil
}

// GetBlock implements the Store interface.
func (s *InmemStore) GetBlock(index int) (*Block, error) {
	res, ok := s.blockCache.Get(index)
	if !ok {
		return nil, cm.NewStoreErr("BlockCache", cm.KeyNotFound, strconv.Itoa(index))
	}
	return res.(*Block), nil
}

// SetBlock implements the Store interface.
func (s *InmemStore) SetBlock(block *Block) error {
	index := block.Index()
	_, err := s.GetBlock(index)
	if err != nil && !cm.IsStore(err, cm.KeyNotFound) {
		return err
	}
	s.blockCache.Add(index, block)
	if index > s.lastBlock {
		s.lastBlock = index
	}
	return nil
}

// LastBlockIndex implements the Store interface.
func (s *InmemStore) LastBlockIndex() int {
	return s.lastBlock
}

// GetFrame implements the Store interface.
func (s *InmemStore) GetFrame(index int) (*Frame, error) {
	res, ok := s.frameCache.Get(index)
	if !ok {
		return nil, cm.NewStoreErr("FrameCache", cm.KeyNotFound, strconv.Itoa(index))
	}
	return res.(*Frame), nil
}

// SetFrame implements the Store interface.
func (s *InmemStore) SetFrame(frame *Frame) error {
	index := frame.Round
	_, err := s.GetFrame(index)
	if err != nil && !cm.IsStore(err, cm.KeyNotFound) {
		return err
	}
	s.frameCache.Add(index, frame)
	return nil
}

// Reset implements the Store interface.
func (s *InmemStore) Reset(frame *Frame) error {
	//Clear all caches
	s.peerSetCache = NewPeerSetCache()
	s.eventCache = cm.NewLRU(s.cacheSize, nil)
	s.roundCache = cm.NewLRU(s.cacheSize, nil)
	s.blockCache = cm.NewLRU(s.cacheSize, nil)
	s.frameCache = cm.NewLRU(s.cacheSize, nil)
	s.participantEventsCache = NewParticipantEventsCache(s.cacheSize)
	s.roots = make(map[string]*Root)
	s.lastRound = -1
	s.lastBlock = -1
	s.consensusCache = cm.NewRollingIndex("ConsensusCache", s.cacheSize)
	s.lastConsensusEvents = map[string]string{}

	//Set Roots from Frame
	s.roots = frame.Roots

	for round, ps := range frame.PeerSets {
		if err := s.SetPeerSet(round, peers.NewPeerSet(ps)); err != nil {
			return err
		}
	}

	//Set Frame
	return s.SetFrame(frame)
}

// Close implements the Store interface.
func (s *InmemStore) Close() error {
	return nil
}

// StorePath implements the Store interface.
func (s *InmemStore) StorePath() string {
	return ""
}
