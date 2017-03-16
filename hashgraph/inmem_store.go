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

type InmemStore struct {
	eventCache        map[string]Event    //hash => event
	consensusEvents   []string            //[index] => hash, in consensus order
	participantEvents map[string][]string //participant => []hash in arrival order
	roundCache        map[int]RoundInfo   //number => RoundInfo
}

func NewInmemStore(participants []string) *InmemStore {
	participantEvents := make(map[string][]string)
	for _, p := range participants {
		participantEvents[p] = []string{}
	}
	return &InmemStore{
		eventCache:        make(map[string]Event),
		consensusEvents:   []string{},
		participantEvents: participantEvents,
		roundCache:        make(map[int]RoundInfo),
	}
}

func (s *InmemStore) GetEvent(key string) (Event, error) {
	res, ok := s.eventCache[key]
	if !ok {
		return Event{}, ErrKeyNotFound
	}
	return res, nil
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
	s.eventCache[key] = event

	return nil
}

func (s *InmemStore) ParticipantEvents(participant string) ([]string, error) {
	events, ok := s.participantEvents[participant]
	if !ok {
		return []string{}, ErrKeyNotFound
	}
	return events, nil
}

func (s *InmemStore) addParticpantEvent(participant string, hash string) error {
	participantEvents, err := s.ParticipantEvents(participant)
	if err != nil && err != ErrKeyNotFound {
		return err
	}
	participantEvents = append(participantEvents, hash)
	s.participantEvents[participant] = participantEvents
	return nil
}

func (s *InmemStore) Known() map[string]int {
	known := make(map[string]int)
	for p, evs := range s.participantEvents {
		known[p] = len(evs)
	}
	return known
}

func (s *InmemStore) ConsensusEvents() []string {
	return s.consensusEvents
}

func (s *InmemStore) AddConsensusEvent(key string) error {
	s.consensusEvents = append(s.consensusEvents, key)
	return nil
}

func (s *InmemStore) GetRound(r int) (RoundInfo, error) {
	res, ok := s.roundCache[r]
	if !ok {
		return *NewRoundInfo(), ErrKeyNotFound
	}
	return res, nil
}

func (s *InmemStore) SetRound(r int, round RoundInfo) error {
	_, err := s.GetRound(r)
	if err != nil && err != ErrKeyNotFound {
		return err
	}
	s.roundCache[r] = round
	return nil
}

func (s *InmemStore) Rounds() int {
	return len(s.roundCache)
}

func (s *InmemStore) RoundWitnesses(r int) []string {
	round, err := s.GetRound(r)
	if err != nil {
		return []string{}
	}
	return round.Witnesses()
}

func (s *InmemStore) Close() error {
	return nil
}
