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
	"sort"
	"time"
)

type Key struct {
	x string
	y string
}
type Couple struct {
	by   string
	from string
}

type Hashgraph struct {
	Participants          []string            //participant public keys
	Events                map[string]Event    //hash => Event, in arrival order
	EventIndex            []string            //[index] => hash
	Rounds                map[int]RoundInfo   //number => RoundInfo
	Consensus             []string            //[index] => hash, in consensus
	ParticipantEvents     map[string][]string //particpant => []hash in arrival order
	LastConsensusEvent    *int                //index of last event which was assigned a round-received
	LastConsensusRound    *int                //index of last round where the fame of all witnesses has been decided
	ConsensusTransactions int                 //number of consensus transactions

	ancestorCache           map[Key]bool
	selfAncestorCache       map[Key]bool
	forkCache               map[Key]bool
	oldestSelfAncestorCache map[Key]string
	stronglySeeCache        map[Key]bool
	parentRoundCache        map[string]int
	roundCache              map[string]int
}

func NewHashgraph(participants []string) Hashgraph {
	participantEvents := make(map[string][]string)
	for _, p := range participants {
		participantEvents[p] = []string{}
	}
	return Hashgraph{
		Participants:            participants,
		Events:                  make(map[string]Event),
		Rounds:                  make(map[int]RoundInfo),
		ParticipantEvents:       participantEvents,
		ancestorCache:           make(map[Key]bool),
		selfAncestorCache:       make(map[Key]bool),
		forkCache:               make(map[Key]bool),
		oldestSelfAncestorCache: make(map[Key]string),
		stronglySeeCache:        make(map[Key]bool),
		parentRoundCache:        make(map[string]int),
		roundCache:              make(map[string]int),
	}
}

func (h *Hashgraph) SuperMajority() int {
	return 2*len(h.Participants)/3 + 1
}

//true if y is an ancestor of x
func (h *Hashgraph) Ancestor(x, y string) bool {
	if c, ok := h.ancestorCache[Key{x, y}]; ok {
		return c
	}
	a := h.ancestor(x, y)
	h.ancestorCache[Key{x, y}] = a
	return a
}

func (h *Hashgraph) ancestor(x, y string) bool {

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
	return h.Ancestor(ex.SelfParent(), y) || h.Ancestor(ex.OtherParent(), y)
}

//true if y is a self-ancestor of x
func (h *Hashgraph) SelfAncestor(x, y string) bool {
	if c, ok := h.selfAncestorCache[Key{x, y}]; ok {
		return c
	}
	a := h.selfAncestor(x, y)
	h.selfAncestorCache[Key{x, y}] = a
	return a
}

func (h *Hashgraph) selfAncestor(x, y string) bool {
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
	return h.SelfAncestor(ex.SelfParent(), y)
}

//true if x detects a fork under y
func (h *Hashgraph) DetectFork(x, y string) bool {
	if c, ok := h.forkCache[Key{x, y}]; ok {
		return c
	}
	f := h.detectFork(x, y)
	return f
}

func (h *Hashgraph) detectFork(x, y string) bool {
	if x == "" || y == "" {
		return false
	}

	if !h.Ancestor(x, y) {
		return false
	}

	ex, ok := h.Events[x]
	if !ok {
		return false
	}
	ey, ok := h.Events[y]
	if !ok {
		return false
	}

	//true if x's self-parent detects a fork under y
	if h.DetectFork(ex.SelfParent(), y) {
		return true
	}

	//If there is a fork, then the information must be contained
	//in x's other-parent
	//Also, the information cannot be in any of x's self-parent's ancestors
	//so the set of events to look through is bounded
	//We look for ancestors of x's other-parent which are not self parents
	//of y's self-descendant and which are not ancestors of x's self-parent
	whistleblower := ex.OtherParent()
	yCreator := ey.Creator()
	var yDescendant string
	if ex.Creator() == yCreator {
		yDescendant = x
	} else {
		yDescendant = y
	}
	stopper := ex.SelfParent()

	return h.blowWhistle(whistleblower, yDescendant, yCreator, stopper)
}

func (h *Hashgraph) blowWhistle(whistleblower string, yDescendant string, yCreator string, stopper string) bool {
	if whistleblower == "" {
		return false
	}
	ex, ok := h.Events[whistleblower]
	if !ok {
		return false
	}
	if h.Ancestor(stopper, whistleblower) {
		return false
	}
	if ex.Creator() == yCreator &&
		!(h.selfAncestor(yDescendant, whistleblower) || h.selfAncestor(whistleblower, yDescendant)) {
		return true
	}

	return h.blowWhistle(ex.SelfParent(), yDescendant, yCreator, stopper) ||
		h.blowWhistle(ex.OtherParent(), yDescendant, yCreator, stopper)
}

//true if x sees y
func (h *Hashgraph) See(x, y string) bool {
	return h.Ancestor(x, y) && !h.DetectFork(x, y)
}

//oldest self-ancestor of x to see y
func (h *Hashgraph) OldestSelfAncestorToSee(x, y string) string {
	if c, ok := h.oldestSelfAncestorCache[Key{x, y}]; ok {
		return c
	}
	res := h.oldestSelfAncestorToSee(x, y)
	h.oldestSelfAncestorCache[Key{x, y}] = res
	return res
}

func (h *Hashgraph) oldestSelfAncestorToSee(x, y string) string {
	if !h.See(x, y) {
		return ""
	}
	ex, _ := h.Events[x]
	a := h.OldestSelfAncestorToSee(ex.SelfParent(), y)
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
	sentinels[ex.Creator()] = true
	if x == y {
		return
	}

	h.MapSentinels(ex.SelfParent(), y, sentinels)
	h.MapSentinels(ex.OtherParent(), y, sentinels)
}

//true if x strongly sees y
func (h *Hashgraph) StronglySee(x, y string) bool {
	if c, ok := h.stronglySeeCache[Key{x, y}]; ok {
		return c
	}
	ss := h.stronglySee(x, y)
	h.stronglySeeCache[Key{x, y}] = ss
	return ss
}

func (h *Hashgraph) stronglySee(x, y string) bool {
	sentinels := make(map[string]bool)
	for _, p := range h.Participants {
		sentinels[p] = false
	}

	h.MapSentinels(x, y, sentinels)

	c := 0
	for _, v := range sentinels {
		if v {
			c++
		}
	}
	return c >= h.SuperMajority()
}

//max of parent rounds
func (h *Hashgraph) ParentRound(x string) int {
	if c, ok := h.parentRoundCache[x]; ok {
		return c
	}
	pr := h.parentRound(x)
	h.parentRoundCache[x] = pr
	return pr
}

func (h *Hashgraph) parentRound(x string) int {
	if x == "" {
		return -1
	}
	ex, ok := h.Events[x]
	if !ok {
		return -1
	}
	if ex.SelfParent() == "" && ex.OtherParent() == "" {
		return 0
	}
	if _, ok := h.Events[ex.SelfParent()]; !ok {
		return 0
	}
	if _, ok := h.Events[ex.OtherParent()]; !ok {
		return 0
	}
	spRound := h.Round(ex.SelfParent())
	opRound := h.Round(ex.OtherParent())

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
	if ex.SelfParent() == "" {
		return true
	}
	_, ok = h.Events[ex.SelfParent()]
	if !ok {
		return false
	}
	return h.Round(x) > h.Round(ex.SelfParent())
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
		if h.StronglySee(x, w) {
			c++
		}
	}

	return c >= h.SuperMajority()
}

func (h *Hashgraph) Round(x string) int {
	if c, ok := h.roundCache[x]; ok {
		return c
	}
	r := h.round(x)
	h.roundCache[x] = r
	return r
}

func (h *Hashgraph) round(x string) int {
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

func (h *Hashgraph) InsertEvent(event Event) error {
	//verify signature
	if ok, err := event.Verify(); !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("Invalid signature")
	}

	if err := h.FromParentsLatest(event); err != nil {
		return fmt.Errorf("ERROR: %s", err)
	}

	hash := event.Hex()
	h.Events[hash] = event
	h.EventIndex = append(h.EventIndex, hash)

	creator := event.Creator()
	h.ParticipantEvents[creator] = append(h.ParticipantEvents[creator], hash)

	return nil
}

//true if parents are last known events of respective creators
func (h *Hashgraph) FromParentsLatest(event Event) error {
	selfParent, otherParent := event.SelfParent(), event.OtherParent()
	creator := event.Creator()
	if selfParent == "" && otherParent == "" &&
		len(h.ParticipantEvents[creator]) == 0 {
		return nil
	}
	selfParentEvent, selfParentExists := h.Events[selfParent]
	if !selfParentExists {
		return fmt.Errorf("Self-parent not known (%s)", selfParent)
	}
	if selfParentEvent.Creator() != creator {
		return fmt.Errorf("Self-parent has different creator")
	}

	_, otherParentExists := h.Events[otherParent]
	if !otherParentExists {
		return fmt.Errorf("Other-parent not known")
	}

	selfParentLegit := selfParent == h.ParticipantEvents[creator][len(h.ParticipantEvents[creator])-1]
	if !selfParentLegit {
		return fmt.Errorf("Self-parent not last known event by creator")
	}

	return nil
}

func (h *Hashgraph) eventLoopStart() int {
	if h.LastConsensusEvent != nil {
		if *h.LastConsensusEvent < len(h.EventIndex)-1 {
			return *h.LastConsensusEvent + 1
		}
		return *h.LastConsensusEvent
	}
	return 0
}

func (h *Hashgraph) DivideRounds() {
	for i := h.eventLoopStart(); i < len(h.EventIndex); i++ {
		hash := h.EventIndex[i]
		roundNumber := h.Round(hash)
		witness := h.Witness(hash)
		roundInfo, _ := h.Rounds[roundNumber]
		roundInfo.AddEvent(hash, witness)
		h.Rounds[roundNumber] = roundInfo
	}
}

func (h *Hashgraph) roundLoopStart() int {
	if h.LastConsensusRound != nil {
		return *h.LastConsensusRound + 1
	}
	return 0
}

//decide if witnesses are famous
func (h *Hashgraph) DecideFame() {
	votes := make(map[string](map[string]bool)) //[x][y]=>vote(x,y)
	for i := h.roundLoopStart(); i < len(h.Rounds)-1; i++ {
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
							if h.StronglySee(y, w) {
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
	for _, x := range h.EventIndex[h.eventLoopStart():] {
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

			//update last consensus round if necessary
			if h.LastConsensusRound == nil || i > *h.LastConsensusRound {
				h.setLastConsensusRound(i)
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
				ex, _ := h.Events[x]
				ex.SetRoundReceived(i)

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

func (h *Hashgraph) setLastConsensusRound(i int) {
	if h.LastConsensusRound == nil {
		h.LastConsensusRound = new(int)
	}
	*h.LastConsensusRound = i
}

func (h *Hashgraph) FindOrder() {
	h.DecideRoundReceived()

	newConsensusEvents := []Event{}
	for xind, x := range h.EventIndex[h.eventLoopStart():] {
		ex, _ := h.Events[x]
		if ex.roundReceived != nil {
			newConsensusEvents = append(newConsensusEvents, ex)
			if h.LastConsensusEvent == nil || xind > *h.LastConsensusEvent {
				h.setLastConsensusEvent(xind)
			}
		}
	}

	sorter := NewConsensusSorter(newConsensusEvents)
	sort.Sort(sorter)

	for _, e := range newConsensusEvents {
		h.Consensus = append(h.Consensus, e.Hex())
		h.ConsensusTransactions += len(e.Transactions())
	}
}

func (h *Hashgraph) setLastConsensusEvent(i int) {
	if h.LastConsensusEvent == nil {
		h.LastConsensusEvent = new(int)
	}
	*h.LastConsensusEvent = i
}

func (h *Hashgraph) MedianTimestamp(eventHashes []string) time.Time {
	events := []Event{}
	for _, x := range eventHashes {
		events = append(events, h.Events[x])
	}
	sort.Sort(ByTimestamp(events))
	return events[len(events)/2].Body.Timestamp
}

//number of events per participants
func (h *Hashgraph) Known() map[string]int {
	known := make(map[string]int)
	for p, evs := range h.ParticipantEvents {
		known[p] = len(evs)
	}
	return known
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
