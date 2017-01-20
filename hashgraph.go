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

import (
	"fmt"
	"math"
	"reflect"
)

type Hashgraph struct {
	Participants []string          //particant public keys
	Events       map[string]Event  //hash => event
	EventIndex   []string          //[index] => hash
	Rounds       map[int]RoundInfo //number => RoundInfo
}

func NewHashgraph(participants []string) Hashgraph {
	return Hashgraph{
		Participants: participants,
		Events:       make(map[string]Event),
		Rounds:       make(map[int]RoundInfo),
	}
}

func (h *Hashgraph) SuperMajority() int {
	return 2*len(h.Participants)/3 + 1
}

//true if y is an ancestor of x
func (h *Hashgraph) Ancestor(x, y string) bool {
	if x == "" {
		return false
	}
	ex, ok := h.Events[x]
	if !ok {
		return false
	}
	if x == y {
		return true
	}
	_, ok = h.Events[y]
	if !ok {
		return false
	}
	return h.Ancestor(ex.Body.Parents[0], y) || h.Ancestor(ex.Body.Parents[1], y)
}

//true if y is a self-ancestor of x
func (h *Hashgraph) SelfAncestor(x, y string) bool {
	if x == "" {
		return false
	}
	ex, ok := h.Events[x]
	if !ok {
		return false
	}
	if x == y {
		return true
	}
	_, ok = h.Events[y]
	if !ok {
		return false
	}
	return h.SelfAncestor(ex.Body.Parents[0], y)
}

//true if x detects a fork under y. also returns the hash of the event which
//is not a self-ancestor of y and caused the fork
func (h *Hashgraph) DetectFork(x, y string) bool {
	if x == "" || y == "" {
		return false
	}
	_, ok := h.Events[x]
	if !ok {
		return false
	}
	ey, ok := h.Events[y]
	if !ok {
		return false
	}

	//filter events satisfying the following criteria:
	//- same creator as y
	//- ancestors of x
	filteredEvents := []Event{}
	for hash, event := range h.Events {
		if reflect.DeepEqual(event.Body.Creator, ey.Body.Creator) &&
			h.Ancestor(x, hash) {
			filteredEvents = append(filteredEvents, event)
		}
	}
	for i := 0; i < len(filteredEvents)-1; i++ {
		a := filteredEvents[i].Hex()
		for j := i + 1; j < len(filteredEvents); j++ {
			b := filteredEvents[j].Hex()
			if !((h.SelfAncestor(a, b)) || h.SelfAncestor(b, a)) {
				return true
			}
		}
	}
	return false
}

//true if x sees y
func (h *Hashgraph) See(x, y string) bool {
	return h.Ancestor(x, y) && !h.DetectFork(x, y)
}

//participants in x's ancestry that see y
func (h *Hashgraph) MapSentinels(x, y string, sentinels map[string]bool) {
	if x == "" {
		return
	}
	ex, ok := h.Events[x]
	if !ok {
		return
	}
	if !h.See(x, y) {
		return
	}
	sentinels[fmt.Sprintf("0x%X", ex.Body.Creator)] = true
	if x == y {
		return
	}

	h.MapSentinels(ex.Body.Parents[0], y, sentinels)
	h.MapSentinels(ex.Body.Parents[1], y, sentinels)
}

//true if x strongly sees y
func (h *Hashgraph) StronglySee(x, y string) (bool, int) {
	sentinels := make(map[string]bool)
	for i := 0; i < len(h.Participants); i++ {
		sentinels[h.Participants[i]] = false
	}

	h.MapSentinels(x, y, sentinels)

	c := 0
	for _, v := range sentinels {
		if v {
			c++
		}
	}
	return c >= h.SuperMajority(), c
}

//max of parent rounds
func (h *Hashgraph) ParentRound(x string) int {
	if x == "" {
		return -1
	}
	ex, ok := h.Events[x]
	if !ok {
		return -1
	}
	if ex.Body.Parents[0] == "" && ex.Body.Parents[1] == "" {
		return 0
	}
	sp, ok := h.Events[ex.Body.Parents[0]]
	if !ok {
		return 0
	}
	op, ok := h.Events[ex.Body.Parents[1]]
	if !ok {
		return 0
	}

	if sp.round > op.round {
		return sp.round
	}
	return op.round
}

//true if x is a witness (first event of a round for the owner)
func (h *Hashgraph) Witness(x string) bool {
	if x == "" {
		return false
	}
	ex, ok := h.Events[x]
	if !ok {
		return false
	}
	if ex.Body.Parents[0] == "" {
		return true
	}
	_, ok = h.Events[ex.Body.Parents[0]]
	if !ok {
		return false
	}
	return h.Round(x) > h.Round(ex.Body.Parents[0])
}

//true if round of x should be incremented
func (h *Hashgraph) RoundInc(x string) bool {
	if x == "" {
		return false
	}

	parentRound := h.ParentRound(x)
	if parentRound < 0 {
		return false
	}

	if len(h.Rounds) < parentRound+1 {
		return false
	}

	roundWitnesses := h.Rounds[parentRound].Witnesses
	c := 0
	for i := 0; i < len(roundWitnesses); i++ {
		if ss, _ := h.StronglySee(x, roundWitnesses[i]); ss {
			c++
		}
	}

	return c >= h.SuperMajority()
}

func (h *Hashgraph) Round(x string) int {
	round := h.ParentRound(x)
	if h.RoundInc(x) {
		round++
	}
	return round
}

//round(x) - round(y)
func (h *Hashgraph) RoundDiff(x, y string) (int, error) {
	if x == "" {
		return math.MinInt64, fmt.Errorf("x is empty")
	}
	if y == "" {
		return math.MinInt64, fmt.Errorf("y is empty")
	}
	_, ok := h.Events[x]
	if !ok {
		return math.MinInt64, fmt.Errorf("event %s not found", x)
	}
	_, ok = h.Events[y]
	if !ok {
		return math.MinInt64, fmt.Errorf("event %s not found", y)
	}

	xRound := h.Round(x)
	if xRound < 0 {
		return math.MinInt64, fmt.Errorf("event %s has negative round", x)
	}
	yRound := h.Round(y)
	if yRound < 0 {
		return math.MinInt64, fmt.Errorf("event %s has negative round", y)
	}

	return xRound - yRound, nil
}

func (h *Hashgraph) InsertEvent(event Event) {
	hash := event.Hex()
	h.Events[hash] = event
	h.EventIndex = append(h.EventIndex, hash)
}

func (h *Hashgraph) DivideRounds() {
	h.Rounds = make(map[int]RoundInfo)
	for i := 0; i < len(h.EventIndex); i++ {
		hash := h.EventIndex[i]
		roundNumber := h.Round(hash)
		witness := h.Witness(hash)
		roundInfo, _ := h.Rounds[roundNumber]
		roundInfo.AddEvent(hash, witness)
		h.Rounds[roundNumber] = roundInfo
	}
}
