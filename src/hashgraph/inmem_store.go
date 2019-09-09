package hashgraph

import (
	"strconv"

	cm "github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
)

// InmemStore ...
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

// NewInmemStore ...
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
		roots:               make(map[string]*Root),
		lastRound:           -1,
		lastBlock:           -1,
		lastConsensusEvents: map[string]string{},
	}
	return store
}

// CacheSize ...
func (s *InmemStore) CacheSize() int {
	return s.cacheSize
}

// GetPeerSet ...
func (s *InmemStore) GetPeerSet(round int) (*peers.PeerSet, error) {
	return s.peerSetCache.Get(round)
}

//SetPeerSet updates the peerSetCache and participantEventsCache
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

// GetAllPeerSets ...
func (s *InmemStore) GetAllPeerSets() (map[int][]*peers.Peer, error) {
	return s.peerSetCache.GetAll()
}

// FirstRound ...
func (s *InmemStore) FirstRound(id uint32) (int, bool) {
	return s.peerSetCache.FirstRound(id)
}

// RepertoireByPubKey ...
func (s *InmemStore) RepertoireByPubKey() map[string]*peers.Peer {
	return s.peerSetCache.RepertoireByPubKey()
}

// RepertoireByID ...
func (s *InmemStore) RepertoireByID() map[uint32]*peers.Peer {
	return s.peerSetCache.RepertoireByID()
}

// GetEvent ...
func (s *InmemStore) GetEvent(key string) (*Event, error) {
	res, ok := s.eventCache.Get(key)
	if !ok {
		return nil, cm.NewStoreErr("EventCache", cm.KeyNotFound, key)
	}

	return res.(*Event), nil
}

// SetEvent ...
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

// ParticipantEvents ...
func (s *InmemStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	return s.participantEventsCache.Get(participant, skip)
}

// ParticipantEvent ...
func (s *InmemStore) ParticipantEvent(participant string, index int) (string, error) {
	return s.participantEventsCache.GetItem(participant, index)
}

// LastEventFrom ...
func (s *InmemStore) LastEventFrom(participant string) (last string, err error) {
	last, err = s.participantEventsCache.GetLast(participant)
	return
}

// LastConsensusEventFrom ...
func (s *InmemStore) LastConsensusEventFrom(participant string) (last string, err error) {
	last, _ = s.lastConsensusEvents[participant]
	return
}

// KnownEvents ...
func (s *InmemStore) KnownEvents() map[uint32]int {
	return s.participantEventsCache.Known()
}

// ConsensusEvents ...
func (s *InmemStore) ConsensusEvents() []string {
	lastWindow, _ := s.consensusCache.GetLastWindow()
	res := make([]string, len(lastWindow))
	for i, item := range lastWindow {
		res[i] = item.(string)
	}
	return res
}

// ConsensusEventsCount ...
func (s *InmemStore) ConsensusEventsCount() int {
	return s.totConsensusEvents
}

// AddConsensusEvent ...
func (s *InmemStore) AddConsensusEvent(event *Event) error {
	s.consensusCache.Set(event.Hex(), s.totConsensusEvents)
	s.totConsensusEvents++
	s.lastConsensusEvents[event.Creator()] = event.Hex()
	return nil
}

// GetRound ...
func (s *InmemStore) GetRound(r int) (*RoundInfo, error) {
	res, ok := s.roundCache.Get(r)
	if !ok {
		return nil, cm.NewStoreErr("RoundCache", cm.KeyNotFound, strconv.Itoa(r))
	}
	return res.(*RoundInfo), nil
}

// SetRound ...
func (s *InmemStore) SetRound(r int, round *RoundInfo) error {
	s.roundCache.Add(r, round)
	if r > s.lastRound {
		s.lastRound = r
	}
	return nil
}

// LastRound ...
func (s *InmemStore) LastRound() int {
	return s.lastRound
}

// RoundWitnesses ...
func (s *InmemStore) RoundWitnesses(r int) []string {
	round, err := s.GetRound(r)
	if err != nil {
		return []string{}
	}
	return round.Witnesses()
}

// RoundEvents ...
func (s *InmemStore) RoundEvents(r int) int {
	round, err := s.GetRound(r)
	if err != nil {
		return 0
	}
	return len(round.CreatedEvents)
}

// GetRoot ...
func (s *InmemStore) GetRoot(participant string) (*Root, error) {
	res, ok := s.roots[participant]
	if !ok {
		return nil, cm.NewStoreErr("RootCache", cm.KeyNotFound, participant)
	}
	return res, nil
}

// GetBlock ...
func (s *InmemStore) GetBlock(index int) (*Block, error) {
	res, ok := s.blockCache.Get(index)
	if !ok {
		return nil, cm.NewStoreErr("BlockCache", cm.KeyNotFound, strconv.Itoa(index))
	}
	return res.(*Block), nil
}

// SetBlock ...
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

// LastBlockIndex ...
func (s *InmemStore) LastBlockIndex() int {
	return s.lastBlock
}

// GetFrame ...
func (s *InmemStore) GetFrame(index int) (*Frame, error) {
	res, ok := s.frameCache.Get(index)
	if !ok {
		return nil, cm.NewStoreErr("FrameCache", cm.KeyNotFound, strconv.Itoa(index))
	}
	return res.(*Frame), nil
}

// SetFrame ...
func (s *InmemStore) SetFrame(frame *Frame) error {
	index := frame.Round
	_, err := s.GetFrame(index)
	if err != nil && !cm.IsStore(err, cm.KeyNotFound) {
		return err
	}
	s.frameCache.Add(index, frame)
	return nil
}

// Reset ...
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

// Close ...
func (s *InmemStore) Close() error {
	return nil
}

// StorePath ...
func (s *InmemStore) StorePath() string {
	return ""
}
