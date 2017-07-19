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

	"github.com/Sirupsen/logrus"

	"github.com/babbleio/babble/common"
)

type Hashgraph struct {
	Participants            map[string]int //[public key] => id
	ReverseParticipants     map[int]string //[id] => public key
	Store                   Store          //Persistent store of Events and Rounds
	UndeterminedEvents      []string       //[index] => hash
	LastConsensusRound      *int           //index of last round where the fame of all witnesses has been decided
	LastCommitedRoundEvents int            //number of events in round before LastConsensusRound
	ConsensusTransactions   int            //number of consensus transactions
	PendingLoadedEvents     int            //number of loaded events that are not yet committed
	commitCh                chan []Event   //channel for committing events
	topologicalIndex        int            //counter used to order events in topological order

	ancestorCache           *common.LRU
	selfAncestorCache       *common.LRU
	oldestSelfAncestorCache *common.LRU
	stronglySeeCache        *common.LRU
	parentRoundCache        *common.LRU
	roundCache              *common.LRU

	logger *logrus.Logger
}

func NewHashgraph(participants map[string]int, store Store, commitCh chan []Event, logger *logrus.Logger) Hashgraph {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	reverseParticipants := make(map[int]string)
	for pk, id := range participants {
		reverseParticipants[id] = pk
	}

	cacheSize := store.CacheSize()
	return Hashgraph{
		Participants:            participants,
		ReverseParticipants:     reverseParticipants,
		Store:                   store,
		commitCh:                commitCh,
		ancestorCache:           common.NewLRU(cacheSize, nil),
		selfAncestorCache:       common.NewLRU(cacheSize, nil),
		oldestSelfAncestorCache: common.NewLRU(cacheSize, nil),
		stronglySeeCache:        common.NewLRU(cacheSize, nil),
		parentRoundCache:        common.NewLRU(cacheSize, nil),
		roundCache:              common.NewLRU(cacheSize, nil),
		logger:                  logger,
	}
}

func (h *Hashgraph) SuperMajority() int {
	return 2*len(h.Participants)/3 + 1
}

//true if y is an ancestor of x
func (h *Hashgraph) Ancestor(x, y string) bool {
	if c, ok := h.ancestorCache.Get(Key{x, y}); ok {
		return c.(bool)
	}
	a := h.ancestor(x, y)
	h.ancestorCache.Add(Key{x, y}, a)
	return a
}

func (h *Hashgraph) ancestor(x, y string) bool {
	if x == "" {
		return false
	}
	if x == y {
		return true
	}

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false
	}

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false
	}
	eyCreator := h.Participants[ey.Creator()]

	lastAncestorKnownFromYCreator := ex.lastAncestors[eyCreator].index

	return lastAncestorKnownFromYCreator >= ey.Index()
}

//true if y is a self-ancestor of x
func (h *Hashgraph) SelfAncestor(x, y string) bool {
	if c, ok := h.selfAncestorCache.Get(Key{x, y}); ok {
		return c.(bool)
	}
	a := h.selfAncestor(x, y)
	h.selfAncestorCache.Add(Key{x, y}, a)
	return a
}

func (h *Hashgraph) selfAncestor(x, y string) bool {
	if x == "" {
		return false
	}
	if x == y {
		return true
	}
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false
	}
	exCreator := h.Participants[ex.Creator()]

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false
	}
	eyCreator := h.Participants[ey.Creator()]

	return exCreator == eyCreator && ex.Index() >= ey.Index()
}

//true if x sees y
func (h *Hashgraph) See(x, y string) bool {
	return h.Ancestor(x, y)
	//it is not necessary to detect forks because we assume that with our
	//implementations, no two events can be added by the same creator at the
	//same height (cf InsertEvent)
}

//oldest self-ancestor of x to see y
func (h *Hashgraph) OldestSelfAncestorToSee(x, y string) string {
	if c, ok := h.oldestSelfAncestorCache.Get(Key{x, y}); ok {
		return c.(string)
	}
	res := h.oldestSelfAncestorToSee(x, y)
	h.oldestSelfAncestorCache.Add(Key{x, y}, res)
	return res
}

func (h *Hashgraph) oldestSelfAncestorToSee(x, y string) string {
	ex, _ := h.Store.GetEvent(x)
	ey, _ := h.Store.GetEvent(y)

	a := ey.firstDescendants[h.Participants[ex.Creator()]]

	if a.index <= ex.Index() {
		return a.hash
	}

	return ""
}

//true if x strongly sees y
func (h *Hashgraph) StronglySee(x, y string) bool {
	if c, ok := h.stronglySeeCache.Get(Key{x, y}); ok {
		return c.(bool)
	}
	ss := h.stronglySee(x, y)
	h.stronglySeeCache.Add(Key{x, y}, ss)
	return ss
}

func (h *Hashgraph) stronglySee(x, y string) bool {

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false
	}

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false
	}

	c := 0
	for i := 0; i < len(ex.lastAncestors); i++ {
		if ex.lastAncestors[i].index >= ey.firstDescendants[i].index {
			c++
		}
	}
	return c >= h.SuperMajority()
}

//max of parent rounds
func (h *Hashgraph) ParentRound(x string) int {
	if c, ok := h.parentRoundCache.Get(x); ok {
		return c.(int)
	}
	pr := h.parentRound(x)
	h.parentRoundCache.Add(x, pr)
	return pr
}

func (h *Hashgraph) parentRound(x string) int {
	if x == "" {
		return -1
	}
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return -1
	}
	if ex.SelfParent() == "" && ex.OtherParent() == "" {
		return 0
	}
	if _, err := h.Store.GetEvent(ex.SelfParent()); err != nil {
		return 0
	}
	if _, err := h.Store.GetEvent(ex.OtherParent()); err != nil {
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
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false
	}
	if ex.SelfParent() == "" {
		return true
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

	if h.Store.Rounds() < parentRound+1 {
		return false
	}

	c := 0
	for _, w := range h.Store.RoundWitnesses(parentRound) {
		if h.StronglySee(x, w) {
			c++
		}
	}

	return c >= h.SuperMajority()
}

func (h *Hashgraph) Round(x string) int {
	if c, ok := h.roundCache.Get(x); ok {
		return c.(int)
	}
	r := h.round(x)
	h.roundCache.Add(x, r)
	return r
}

func (h *Hashgraph) round(x string) int {
	round := h.ParentRound(x)

	inc := h.RoundInc(x)

	if inc {
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
		return err
	}

	event.topologicalIndex = h.topologicalIndex
	h.topologicalIndex++

	if err := h.SetWireInfo(&event); err != nil {
		return err
	}

	if err := h.InitEventCoordinates(&event); err != nil {
		return err
	}

	if err := h.Store.SetEvent(event); err != nil {
		return err
	}

	if err := h.UpdateAncestorFirstDescendant(event); err != nil {
		return err
	}

	h.UndeterminedEvents = append(h.UndeterminedEvents, event.Hex())

	if event.IsLoaded() {
		h.PendingLoadedEvents++
	}

	return nil
}

//true if parents are last known events of respective creators
func (h *Hashgraph) FromParentsLatest(event Event) error {
	selfParent, otherParent := event.SelfParent(), event.OtherParent()
	creator := event.Creator()
	creatorKnownEvents, _ := h.Store.Known()[h.Participants[creator]]
	if selfParent == "" && otherParent == "" && creatorKnownEvents == 0 {
		return nil
	}
	selfParentEvent, selfParentError := h.Store.GetEvent(selfParent)
	if selfParentError != nil {
		return fmt.Errorf("Self-parent not known (%s)", selfParent)
	}
	if selfParentEvent.Creator() != creator {
		return fmt.Errorf("Self-parent has different creator")
	}

	if otherParent != "" {
		_, otherParentError := h.Store.GetEvent(otherParent)
		if otherParentError != nil {
			return fmt.Errorf("Other-parent not known (%s)", otherParent)
		}
	}

	lastKnown, err := h.Store.LastFrom(creator)
	if err != nil {
		return err
	}
	selfParentLegit := selfParent == lastKnown
	if !selfParentLegit {
		return fmt.Errorf("Self-parent not last known event by creator")
	}

	return nil
}

//initialize arrays of last ancestors and first descendants
func (h *Hashgraph) InitEventCoordinates(event *Event) error {
	members := len(h.Participants)

	event.firstDescendants = make([]EventCoordinates, members)
	for fakeID := 0; fakeID < members; fakeID++ {
		event.firstDescendants[fakeID] = EventCoordinates{
			index: math.MaxInt64,
		}
	}

	event.lastAncestors = make([]EventCoordinates, members)
	if event.SelfParent() == "" && event.OtherParent() == "" {
		for fakeID := 0; fakeID < members; fakeID++ {
			event.lastAncestors[fakeID] = EventCoordinates{
				index: -1,
			}
		}
	} else if event.SelfParent() == "" {
		otherParent, err := h.Store.GetEvent(event.OtherParent())
		if err != nil {
			return err
		}
		copy(event.lastAncestors[:members], otherParent.lastAncestors)
	} else if event.OtherParent() == "" {
		selfParent, err := h.Store.GetEvent(event.SelfParent())
		if err != nil {
			return err
		}
		copy(event.lastAncestors[:members], selfParent.lastAncestors)
	} else {
		selfParent, err := h.Store.GetEvent(event.SelfParent())
		if err != nil {
			return err
		}
		selfParentLastAncestors := selfParent.lastAncestors

		otherParent, err := h.Store.GetEvent(event.OtherParent())
		if err != nil {
			return err
		}
		otherParentLastAncestors := otherParent.lastAncestors

		copy(event.lastAncestors[:members], selfParentLastAncestors)
		for i := 0; i < members; i++ {
			if event.lastAncestors[i].index < otherParentLastAncestors[i].index {
				event.lastAncestors[i].index = otherParentLastAncestors[i].index
				event.lastAncestors[i].hash = otherParentLastAncestors[i].hash
			}
		}
	}

	index := event.Index()

	creator := event.Creator()
	fakeCreatorID, ok := h.Participants[creator]
	if !ok {
		return fmt.Errorf("Could not find fake creator id")
	}
	hash := event.Hex()

	event.firstDescendants[fakeCreatorID] = EventCoordinates{index: index, hash: hash}
	event.lastAncestors[fakeCreatorID] = EventCoordinates{index: index, hash: hash}

	return nil
}

//update first decendant of each last ancestor to point to event
func (h *Hashgraph) UpdateAncestorFirstDescendant(event Event) error {
	fakeCreatorID, ok := h.Participants[event.Creator()]
	if !ok {
		return fmt.Errorf("Could not find creator fake id (%s)", event.Creator())
	}
	index := event.Index()
	hash := event.Hex()

	for i := 0; i < len(event.lastAncestors); i++ {
		ah := event.lastAncestors[i].hash
		for ah != "" {
			a, err := h.Store.GetEvent(ah)
			if err != nil {
				return err
			}
			if a.firstDescendants[fakeCreatorID].index == math.MaxInt64 {
				a.firstDescendants[fakeCreatorID] = EventCoordinates{index: index, hash: hash}
				if err := h.Store.SetEvent(a); err != nil {
					return err
				}
				ah = a.SelfParent()
			} else {
				break
			}
		}
	}

	return nil
}

func (h *Hashgraph) SetWireInfo(event *Event) error {
	selfParentIndex := -1
	otherParentCreatorID := -1
	otherParentIndex := -1

	if event.SelfParent() != "" {
		selfParent, err := h.Store.GetEvent(event.SelfParent())
		if err != nil {
			return err
		}
		selfParentIndex = selfParent.Index()
	}

	if event.OtherParent() != "" {
		otherParent, err := h.Store.GetEvent(event.OtherParent())
		if err != nil {
			return err
		}
		otherParentCreatorID = h.Participants[otherParent.Creator()]
		otherParentIndex = otherParent.Index()
	}

	event.SetWireInfo(selfParentIndex,
		otherParentCreatorID,
		otherParentIndex,
		h.Participants[event.Creator()])

	return nil
}

func (h *Hashgraph) ReadWireInfo(wevent WireEvent) (*Event, error) {
	selfParent := ""
	otherParent := ""
	var err error

	creator := h.ReverseParticipants[wevent.Body.CreatorID]
	creatorBytes, err := hex.DecodeString(creator[2:])
	if err != nil {
		return nil, err
	}

	if wevent.Body.SelfParentIndex >= 0 {
		selfParent, err = h.Store.ParticipantEvent(creator, wevent.Body.SelfParentIndex)
		if err != nil {
			return nil, err
		}
	}
	if wevent.Body.OtherParentIndex >= 0 {
		otherParentCreator := h.ReverseParticipants[wevent.Body.OtherParentCreatorID]
		otherParent, err = h.Store.ParticipantEvent(otherParentCreator, wevent.Body.OtherParentIndex)
		if err != nil {
			return nil, err
		}
	}

	body := EventBody{
		Transactions: wevent.Body.Transactions,
		Parents:      []string{selfParent, otherParent},
		Creator:      creatorBytes,

		Timestamp:            wevent.Body.Timestamp,
		Index:                wevent.Body.Index,
		selfParentIndex:      wevent.Body.SelfParentIndex,
		otherParentCreatorID: wevent.Body.OtherParentCreatorID,
		otherParentIndex:     wevent.Body.OtherParentIndex,
		creatorID:            wevent.Body.CreatorID,
	}

	event := &Event{
		Body: body,
		R:    wevent.R,
		S:    wevent.S,
	}

	return event, nil
}

func (h *Hashgraph) DivideRounds() error {
	for _, hash := range h.UndeterminedEvents {
		roundNumber := h.Round(hash)
		witness := h.Witness(hash)
		roundInfo, err := h.Store.GetRound(roundNumber)
		if err != nil && err != ErrKeyNotFound {
			return err
		}
		roundInfo.AddEvent(hash, witness)
		err = h.Store.SetRound(roundNumber, roundInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Hashgraph) fameLoopStart() int {
	if h.LastConsensusRound != nil {
		return *h.LastConsensusRound + 1
	}
	return 0
}

//decide if witnesses are famous
func (h *Hashgraph) DecideFame() error {
	votes := make(map[string](map[string]bool)) //[x][y]=>vote(x,y)
	for i := h.fameLoopStart(); i < h.Store.Rounds()-1; i++ {
		roundInfo, err := h.Store.GetRound(i)
		if err != nil {
			return err
		}
		for j := i + 1; j < h.Store.Rounds(); j++ {
			for _, x := range roundInfo.Witnesses() {
				for _, y := range h.Store.RoundWitnesses(j) {
					diff := j - i
					if diff == 1 {
						setVote(votes, y, x, h.See(y, x))
					} else {
						//count votes
						ssWitnesses := []string{}
						for _, w := range h.Store.RoundWitnesses(j - 1) {
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
		if roundInfo.WitnessesDecided() &&
			(h.LastConsensusRound == nil || i > *h.LastConsensusRound) {
			h.setLastConsensusRound(i)
		}
		err = h.Store.SetRound(i, roundInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Hashgraph) setLastConsensusRound(i int) {
	if h.LastConsensusRound == nil {
		h.LastConsensusRound = new(int)
	}
	*h.LastConsensusRound = i

	h.LastCommitedRoundEvents = h.Store.RoundEvents(i - 1)
}

//assign round received and timestamp to all events
func (h *Hashgraph) DecideRoundReceived() error {
	for _, x := range h.UndeterminedEvents {
		r := h.Round(x)
		for i := r + 1; i < h.Store.Rounds(); i++ {
			tr, err := h.Store.GetRound(i)
			if err != nil {
				return err
			}
			//no witnesses are left undecided
			if !tr.WitnessesDecided() {
				continue
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
				ex, err := h.Store.GetEvent(x)
				if err != nil {
					return err
				}
				ex.SetRoundReceived(i)

				t := []string{}
				for _, a := range s {
					t = append(t, h.OldestSelfAncestorToSee(a, x))
				}

				ex.consensusTimestamp = h.MedianTimestamp(t)

				err = h.Store.SetEvent(ex)
				if err != nil {
					return err
				}

				break
			}
		}
	}
	return nil
}

func (h *Hashgraph) FindOrder() error {
	err := h.DecideRoundReceived()
	if err != nil {
		return err
	}

	newConsensusEvents := []Event{}
	newUndeterminedEvents := []string{}
	for _, x := range h.UndeterminedEvents {
		ex, err := h.Store.GetEvent(x)
		if err != nil {
			return err
		}
		if ex.roundReceived != nil {
			newConsensusEvents = append(newConsensusEvents, ex)
		} else {
			newUndeterminedEvents = append(newUndeterminedEvents, x)
		}
	}
	h.UndeterminedEvents = newUndeterminedEvents

	sorter := NewConsensusSorter(newConsensusEvents)
	sort.Sort(sorter)

	for _, e := range newConsensusEvents {
		err := h.Store.AddConsensusEvent(e.Hex())
		if err != nil {
			return err
		}
		h.ConsensusTransactions += len(e.Transactions())
		if e.IsLoaded() {
			h.PendingLoadedEvents--
		}
	}

	if h.commitCh != nil && len(newConsensusEvents) > 0 {
		h.commitCh <- newConsensusEvents
	}

	return nil
}

func (h *Hashgraph) MedianTimestamp(eventHashes []string) time.Time {
	events := []Event{}
	for _, x := range eventHashes {
		ex, _ := h.Store.GetEvent(x)
		events = append(events, ex)
	}
	sort.Sort(ByTimestamp(events))
	return events[len(events)/2].Body.Timestamp
}

func (h *Hashgraph) ConsensusEvents() []string {
	return h.Store.ConsensusEvents()
}

//number of events per participants
func (h *Hashgraph) Known() map[int]int {
	return h.Store.Known()
}

func middleBit(ehex string) bool {
	hash, err := hex.DecodeString(ehex[2:])
	if err != nil {
		fmt.Printf("ERROR decoding hex string: %s\n", err)
	}
	if len(hash) > 0 && hash[len(hash)/2] == 0 {
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
