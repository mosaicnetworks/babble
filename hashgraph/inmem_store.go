package hashgraph

import cm "github.com/babbleio/babble/common"

type InmemStore struct {
	cacheSize              int
	eventCache             *cm.LRU
	roundCache             *cm.LRU
	consensusCache         *cm.RollingIndex
	totConsensusEvents     int
	participantEventsCache *ParticipantEventsCache
	roots                  map[string]Root
}

func NewInmemStore(participants map[string]int, cacheSize int) *InmemStore {
	roots := make(map[string]Root)
	for pk := range participants {
		roots[pk] = Root{
			X:     "",
			Y:     "",
			Index: -1,
			Round: 0,
		}
	}
	return &InmemStore{
		cacheSize:              cacheSize,
		eventCache:             cm.NewLRU(cacheSize, nil),
		roundCache:             cm.NewLRU(cacheSize, nil),
		consensusCache:         cm.NewRollingIndex(cacheSize),
		participantEventsCache: NewParticipantEventsCache(cacheSize, participants),
		roots: roots,
	}
}

func (s *InmemStore) CacheSize() int {
	return s.cacheSize
}

func (s *InmemStore) GetEvent(key string) (Event, error) {
	res, ok := s.eventCache.Get(key)
	if !ok {
		return Event{}, cm.NewStoreErr(cm.KeyNotFound, key)
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

func (s *InmemStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	return s.participantEventsCache.Get(participant, skip)
}

func (s *InmemStore) ParticipantEvent(particant string, index int) (string, error) {
	return s.participantEventsCache.GetItem(particant, index)
}

func (s *InmemStore) LastFrom(participant string) (string, error) {
	last, err := s.participantEventsCache.GetLast(participant)
	if err != nil {
		return "", err
	}
	if last == "" {
		root, ok := s.roots[participant]
		if ok {
			last = root.X
		}
	}
	return last, nil
}

func (s *InmemStore) addParticpantEvent(participant string, hash string, index int) error {
	return s.participantEventsCache.Add(participant, hash, index)
}

func (s *InmemStore) Known() map[int]int {
	return s.participantEventsCache.Known()
}

func (s *InmemStore) ConsensusEvents() []string {
	lastWindow, _ := s.consensusCache.GetLastWindow()
	res := []string{}
	for _, item := range lastWindow {
		res = append(res, item.(string))
	}
	return res
}

func (s *InmemStore) ConsensusEventsCount() int {
	return s.totConsensusEvents
}

func (s *InmemStore) AddConsensusEvent(key string) error {
	s.consensusCache.Add(key, s.totConsensusEvents)
	s.totConsensusEvents++
	return nil
}

func (s *InmemStore) GetRound(r int) (RoundInfo, error) {
	res, ok := s.roundCache.Get(r)
	if !ok {
		return *NewRoundInfo(), ErrKeyNotFound
	}
	return res.(RoundInfo), nil
}

func (s *InmemStore) SetRound(r int, round RoundInfo) error {
	s.roundCache.Add(r, round)
	return nil
}

func (s *InmemStore) Rounds() int {
	return s.roundCache.Len()
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
	res, ok := s.roots[participant]
	if !ok {
		return Root{}, ErrKeyNotFound
	}
	return res, nil
}

func (s *InmemStore) Reset(roots map[string]Root) error {
	s.roots = roots
	s.eventCache = cm.NewLRU(s.cacheSize, nil)
	s.roundCache = cm.NewLRU(s.cacheSize, nil)
	s.consensusCache = cm.NewRollingIndex(s.cacheSize)
	err := s.participantEventsCache.Reset()
	return err
}

func (s *InmemStore) Close() error {
	return nil
}
