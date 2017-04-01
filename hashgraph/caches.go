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

import "github.com/arrivets/babble/common"

type Key struct {
	x string
	y string
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//PARTICIPANT EVENTS CACHE
type ParticipantEventsCache struct {
	size              int
	participantEvents map[string]*common.RollingList
}

func NewParticipantEventsCache(size int, participants []string) *ParticipantEventsCache {
	items := make(map[string]*common.RollingList)
	for _, p := range participants {
		items[p] = common.NewRollingList(size)
	}
	return &ParticipantEventsCache{
		size:              size,
		participantEvents: items,
	}
}

func (pec *ParticipantEventsCache) Get(participant string, skip int) ([]string, error) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		return []string{}, ErrKeyNotFound
	}

	cached, tot := pe.Get()

	if len(cached) == 0 {
		return []string{}, nil
	}

	lastCached := tot - len(cached)
	if lastCached > skip {
		//XXX TODO
		//LOAD REST FROM FILE
		return []string{}, ErrTooLate
	}

	start := skip % pec.size
	if start >= len(cached) {
		return []string{}, nil
	}

	res := []string{}
	for k := start; k < len(cached); k++ {
		res = append(res, cached[k].(string))
	}
	return res, nil
}

func (pec *ParticipantEventsCache) GetLast(participant string) (string, error) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		return "", ErrKeyNotFound
	}
	cached, _ := pe.Get()
	if len(cached) == 0 {
		return "", nil
	}
	last := cached[len(cached)-1]
	return last.(string), nil
}

func (pec *ParticipantEventsCache) Add(participant string, hash string) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		pe = common.NewRollingList(pec.size)
		pec.participantEvents[participant] = pe
	}
	pe.Add(hash)
}

func (pec *ParticipantEventsCache) Known() map[string]int {
	known := make(map[string]int)
	for p, evs := range pec.participantEvents {
		_, tot := evs.Get()
		known[p] = tot
	}
	return known
}
