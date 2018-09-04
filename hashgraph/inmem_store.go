package hashgraph

import (
	"strconv"

	cm "github.com/mosaicnetworks/babble/common"
)

type InmemStore struct {
	cacheSize              int
	participants           map[string]int
	eventCache             *cm.LRU
	roundCache             *cm.LRU
	blockCache             *cm.LRU
	frameCache             *cm.LRU
	consensusCache         *cm.RollingIndex
	totConsensusEvents     int
	participantEventsCache *ParticipantEventsCache
	rootsByParticipant     map[string]Root //[participant] => Root
	rootsBySelfParent      map[string]Root //[Root.SelfParent.Hash] => Root
	lastRound              int
	lastConsensusEvents    map[string]string //[participant] => hex() of last consensus event
	lastBlock              int
}

func NewInmemStore(participants map[string]int, cacheSize int) *InmemStore {
	rootsByParticipant := make(map[string]Root)
	for pk, pid := range participants {
		root := NewBaseRoot(pid)
		rootsByParticipant[pk] = root
	}
	return &InmemStore{
		cacheSize:              cacheSize,
		participants:           participants,
		eventCache:             cm.NewLRU(cacheSize, nil),
		roundCache:             cm.NewLRU(cacheSize, nil),
		blockCache:             cm.NewLRU(cacheSize, nil),
		frameCache:             cm.NewLRU(cacheSize, nil),
		consensusCache:         cm.NewRollingIndex("ConsensusCache", cacheSize),
		participantEventsCache: NewParticipantEventsCache(cacheSize, participants),
		rootsByParticipant:     rootsByParticipant,
		lastRound:              -1,
		lastBlock:              -1,
		lastConsensusEvents:    map[string]string{},
	}
}

func (s *InmemStore) CacheSize() int {
	return s.cacheSize
}

func (s *InmemStore) Participants() (map[string]int, error) {
	return s.participants, nil
}

func (s *InmemStore) RootsBySelfParent() (map[string]Root, error) {
	if s.rootsBySelfParent == nil {
		s.rootsBySelfParent = make(map[string]Root)
		for _, root := range s.rootsByParticipant {
			s.rootsBySelfParent[root.SelfParent.Hash] = root
		}
	}
	return s.rootsBySelfParent, nil
}

func (s *InmemStore) GetEvent(key string) (Event, error) {
	res, ok := s.eventCache.Get(key)
	if !ok {
		return Event{}, cm.NewStoreErr("EventCache", cm.KeyNotFound, key)
	}

	return res.(Event), nil
}

func (s *InmemStore) SetEvent(event Event) error {
	key := event.Hex()
	_, err := s.GetEvent(key)
	if err != nil && !cm.Is(err, cm.KeyNotFound) {
		return err
	}
	if cm.Is(err, cm.KeyNotFound) {
		if err := s.addParticpantEvent(event.Creator(), key, event.Index()); err != nil {
			return err
		}
	}
	s.eventCache.Add(key, event)

	return nil
}

func (s *InmemStore) addParticpantEvent(participant string, hash string, index int) error {
	return s.participantEventsCache.Set(participant, hash, index)
}

func (s *InmemStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	return s.participantEventsCache.Get(participant, skip)
}

func (s *InmemStore) ParticipantEvent(participant string, index int) (string, error) {
	ev, err := s.participantEventsCache.GetItem(participant, index)
	if err != nil {
		root, ok := s.rootsByParticipant[participant]
		if !ok {
			return "", cm.NewStoreErr("InmemStore.Roots", cm.NoRoot, participant)
		}
		if root.SelfParent.Index == index {
			ev = root.SelfParent.Hash
			err = nil
		}
	}
	return ev, err
}

func (s *InmemStore) LastEventFrom(participant string) (last string, isRoot bool, err error) {
	//try to get the last event from this participant
	last, err = s.participantEventsCache.GetLast(participant)

	//if there is none, grab the root
	if err != nil && cm.Is(err, cm.Empty) {
		root, ok := s.rootsByParticipant[participant]
		if ok {
			last = root.SelfParent.Hash
			isRoot = true
			err = nil
		} else {
			err = cm.NewStoreErr("InmemStore.Roots", cm.NoRoot, participant)
		}
	}
	return
}

func (s *InmemStore) LastConsensusEventFrom(participant string) (last string, isRoot bool, err error) {
	//try to get the last consensus event from this participant
	last, ok := s.lastConsensusEvents[participant]
	//if there is none, grab the root
	if !ok {
		root, ok := s.rootsByParticipant[participant]
		if ok {
			last = root.SelfParent.Hash
			isRoot = true
		} else {
			err = cm.NewStoreErr("InmemStore.Roots", cm.NoRoot, participant)
		}
	}
	return
}

func (s *InmemStore) KnownEvents() map[int]int {
	known := s.participantEventsCache.Known()
	for p, pid := range s.participants {
		if known[pid] == -1 {
			root, ok := s.rootsByParticipant[p]
			if ok {
				known[pid] = root.SelfParent.Index
			}
		}
	}
	return known
}

func (s *InmemStore) ConsensusEvents() []string {
	lastWindow, _ := s.consensusCache.GetLastWindow()
	res := make([]string, len(lastWindow))
	for i, item := range lastWindow {
		res[i] = item.(string)
	}
	return res
}

func (s *InmemStore) ConsensusEventsCount() int {
	return s.totConsensusEvents
}

func (s *InmemStore) AddConsensusEvent(event Event) error {
	s.consensusCache.Set(event.Hex(), s.totConsensusEvents)
	s.totConsensusEvents++
	s.lastConsensusEvents[event.Creator()] = event.Hex()
	return nil
}

func (s *InmemStore) GetRound(r int) (RoundInfo, error) {
	res, ok := s.roundCache.Get(r)
	if !ok {
		return *NewRoundInfo(), cm.NewStoreErr("RoundCache", cm.KeyNotFound, strconv.Itoa(r))
	}
	return res.(RoundInfo), nil
}

func (s *InmemStore) SetRound(r int, round RoundInfo) error {
	s.roundCache.Add(r, round)
	if r > s.lastRound {
		s.lastRound = r
	}
	return nil
}

func (s *InmemStore) LastRound() int {
	return s.lastRound
}

func (s *InmemStore) RoundWitnesses(r int) []string {
	round, err := s.GetRound(r)
	if err != nil {
		return []string{}
	}
	return round.Witnesses()
}

func (s *InmemStore) RoundEvents(r int) int {
	round, err := s.GetRound(r)
	if err != nil {
		return 0
	}
	return len(round.Events)
}

func (s *InmemStore) GetRoot(participant string) (Root, error) {
	res, ok := s.rootsByParticipant[participant]
	if !ok {
		return Root{}, cm.NewStoreErr("RootCache", cm.KeyNotFound, participant)
	}
	return res, nil
}

func (s *InmemStore) GetBlock(index int) (Block, error) {
	res, ok := s.blockCache.Get(index)
	if !ok {
		return Block{}, cm.NewStoreErr("BlockCache", cm.KeyNotFound, strconv.Itoa(index))
	}
	return res.(Block), nil
}

func (s *InmemStore) SetBlock(block Block) error {
	index := block.Index()
	_, err := s.GetBlock(index)
	if err != nil && !cm.Is(err, cm.KeyNotFound) {
		return err
	}
	s.blockCache.Add(index, block)
	if index > s.lastBlock {
		s.lastBlock = index
	}
	return nil
}

func (s *InmemStore) LastBlockIndex() int {
	return s.lastBlock
}

func (s *InmemStore) GetFrame(index int) (Frame, error) {
	res, ok := s.frameCache.Get(index)
	if !ok {
		return Frame{}, cm.NewStoreErr("FrameCache", cm.KeyNotFound, strconv.Itoa(index))
	}
	return res.(Frame), nil
}

func (s *InmemStore) SetFrame(frame Frame) error {
	index := frame.Round
	_, err := s.GetFrame(index)
	if err != nil && !cm.Is(err, cm.KeyNotFound) {
		return err
	}
	s.frameCache.Add(index, frame)
	return nil
}

func (s *InmemStore) Reset(roots map[string]Root) error {
	s.rootsByParticipant = roots
	s.rootsBySelfParent = nil
	s.eventCache = cm.NewLRU(s.cacheSize, nil)
	s.roundCache = cm.NewLRU(s.cacheSize, nil)
	s.consensusCache = cm.NewRollingIndex("ConsensusCache", s.cacheSize)
	err := s.participantEventsCache.Reset()
	s.lastRound = -1
	s.lastBlock = -1
	return err
}

func (s *InmemStore) Close() error {
	return nil
}
