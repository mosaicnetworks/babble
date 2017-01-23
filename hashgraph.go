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
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"sort"
	"time"
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

//oldest self-ancestor of x to see y
func (h *Hashgraph) OldestSelfAncestorToSee(x, y string) string {
	if !h.See(x, y) {
		return ""
	}
	ex, _ := h.Events[x]
	a := h.OldestSelfAncestorToSee(ex.Body.Parents[0], y)
	if a == "" {
		return x
	}
	return a
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
	if _, ok := h.Events[ex.Body.Parents[0]]; !ok {
		return 0
	}
	if _, ok := h.Events[ex.Body.Parents[1]]; !ok {
		return 0
	}
	spRound := h.Round(ex.Body.Parents[0])
	opRound := h.Round(ex.Body.Parents[1])

	if spRound > opRound {
		return spRound
	}
	return opRound
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
	for w := range roundWitnesses {
		if ss, _ := h.StronglySee(x, w); ss {
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

//decide if witnesses are famous
func (h *Hashgraph) DecideFame() {
	votes := make(map[string](map[string]bool)) //[x][y]=>vote(x,y)
	for i := 0; i < len(h.Rounds)-1; i++ {
		roundInfo := h.Rounds[i]
		for j := i + 1; j < len(h.Rounds); j++ {
			for x := range roundInfo.Witnesses {
				for y := range h.Rounds[j].Witnesses {
					diff := j - i
					if diff == 1 {
						setVote(votes, y, x, h.See(y, x))
					} else {
						//count votes
						ssWitnesses := []string{}
						for w := range h.Rounds[j-1].Witnesses {
							if ss, _ := h.StronglySee(y, w); ss {
								ssWitnesses = append(ssWitnesses, w)
							}
						}
						yays := 0
						nays := 0
						for _, w := range ssWitnesses {
							if votes[w][x] {
								yays++
							} else {
								nays++
							}
						}
						v := false
						t := nays
						if yays >= nays {
							v = true
							t = yays
						}

						//normal round
						if math.Mod(float64(diff), float64(len(h.Participants))) > 0 {
							if t >= h.SuperMajority() {
								roundInfo.SetFame(x, v)
								break //break out of y loop
							} else {
								setVote(votes, y, x, v)
							}
						} else { //coin round
							if t >= h.SuperMajority() {
								setVote(votes, y, x, v)
							} else {
								setVote(votes, y, x, middleBit(y)) //middle bit of y's hash
							}
						}
					}
				}
			}
		}
		h.Rounds[i] = roundInfo
	}
}

//assign round received and timestamp to all events
func (h *Hashgraph) DecideRoundReceived() {
	for _, x := range h.EventIndex {
		r := h.Round(x)
		for i := r + 1; i < len(h.Rounds); i++ {
			tr, ok := h.Rounds[i]
			if !ok {
				break
			}
			//no witnesses are left undecided
			if !tr.WitnessesDecided() {
				break
			}
			fws := tr.FamousWitnesses()
			//set of famous witnesses that see x
			s := []string{}
			for _, w := range fws {
				if h.See(w, x) {
					s = append(s, w)
				}
			}
			if len(s) > len(fws)/2 {
				fmt.Printf("event %s round received %d\n", x, i)
				ex, _ := h.Events[x]
				ex.roundReceived = i

				t := []string{}
				for _, a := range s {
					t = append(t, h.OldestSelfAncestorToSee(a, x))
				}

				ex.consensusTimestamp = h.MedianTimestamp(t)

				h.Events[x] = ex
				break
			}
		}
	}
}

func (h *Hashgraph) MedianTimestamp(eventHashes []string) time.Time {
	events := []Event{}
	for _, x := range eventHashes {
		events = append(events, h.Events[x])
	}
	sort.Sort(ByTimestamp(events))
	return events[len(events)/2].Body.Timestamp
}

func middleBit(ehex string) bool {
	hash, err := hex.DecodeString(ehex)
	if err == nil {
		fmt.Printf("ERROR decoding hex string: %s\n", err)
	}
	if hash[len(hash)/2] == 0 {
		return false
	}
	return true
}

func setVote(votes map[string]map[string]bool, x, y string, vote bool) {
	if votes[x] == nil {
		votes[x] = make(map[string]bool)
	}
	votes[x][y] = vote
}
