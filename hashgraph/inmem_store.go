/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package hashgraph

import "bitbucket.org/mosaicnet/babble/common"

type InmemStore struct {
	cacheSize              int
	eventCache             *common.LRU
	roundCache             *common.LRU
	consensusCache         *common.RollingList
	participantEventsCache *ParticipantEventsCache
}

func NewInmemStore(participants map[string]int, cacheSize int) *InmemStore {
	return &InmemStore{
		cacheSize:              cacheSize,
		eventCache:             common.NewLRU(cacheSize, nil),
		roundCache:             common.NewLRU(cacheSize, nil),
		consensusCache:         common.NewRollingList(cacheSize),
		participantEventsCache: NewParticipantEventsCache(cacheSize, participants),
	}
}

func (s *InmemStore) CacheSize() int {
	return s.cacheSize
}

func (s *InmemStore) GetEvent(key string) (Event, error) {
	res, ok := s.eventCache.Get(key)
	if !ok {
		return Event{}, ErrKeyNotFound
	}

	return res.(Event), nil
}

func (s *InmemStore) SetEvent(event Event) error {
	key := event.Hex()
	_, err := s.GetEvent(key)
	if err != nil && err != ErrKeyNotFound {
		return err
	}
	if err == ErrKeyNotFound {
		if err := s.addParticpantEvent(event.Creator(), key); err != nil {
			return err
		}
	}
	s.eventCache.Add(key, event)

	return nil
}

func (s *InmemStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	return s.participantEventsCache.Get(participant, skip)
}

func (s *InmemStore) LastFrom(participant string) (string, error) {
	return s.participantEventsCache.GetLast(participant)
}

func (s *InmemStore) addParticpantEvent(participant string, hash string) error {
	s.participantEventsCache.Add(participant, hash)
	return nil
}

func (s *InmemStore) Known() map[int]int {
	return s.participantEventsCache.Known()
}

func (s *InmemStore) ConsensusEvents() []string {
	lastWindow, _ := s.consensusCache.Get()
	res := []string{}
	for _, item := range lastWindow {
		res = append(res, item.(string))
	}
	return res
}

func (s *InmemStore) ConsensusEventsCount() int {
	_, tot := s.consensusCache.Get()
	return tot
}

func (s *InmemStore) AddConsensusEvent(key string) error {
	s.consensusCache.Add(key)
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

func (s *InmemStore) Close() error {
	return nil
}
