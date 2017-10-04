package hashgraph

import (
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Sirupsen/logrus"

	"bitbucket.org/mosaicnet/babble/common"
)

type Hashgraph struct {
	Participants            map[string]int //[public key] => id
	ReverseParticipants     map[int]string //[id] => public key
	Store                   Store          //store of Events and Rounds
	UndeterminedEvents      []string       //[index] => hash
	UndecidedRounds         []int          //queue of Rounds which have undecided witnesses
	LastConsensusRound      *int           //index of last round where the fame of all witnesses has been decided
	LastCommitedRoundEvents int            //number of events in round before LastConsensusRound
	ConsensusTransactions   int            //number of consensus transactions
	PendingLoadedEvents     int            //number of loaded events that are not yet committed
	commitCh                chan []Event   //channel for committing events
	topologicalIndex        int            //counter used to order events in topological order
	superMajority           int

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
		superMajority:           2*len(participants)/3 + 1,
		UndecidedRounds:         []int{0}, //initialize
	}
}

func (h *Hashgraph) SuperMajority() int {
	return h.superMajority
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
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return ""
	}
	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return ""
	}

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

//round: max of parent rounds
//isRoot: true if round is taken from a Root
func (h *Hashgraph) ParentRound(x string) ParentRoundInfo {
	if c, ok := h.parentRoundCache.Get(x); ok {
		return c.(ParentRoundInfo)
	}
	pr := h.parentRound(x)
	h.parentRoundCache.Add(x, pr)
	return pr
}

func (h *Hashgraph) parentRound(x string) ParentRoundInfo {
	res := NewBaseParentRoundInfo()

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return res
	}

	//We are going to need the Root later
	root, err := h.Store.GetRoot(ex.Creator())
	if err != nil {
		return res
	}

	spRound := -1
	spRoot := false
	//If it is the creator's first Event, use the corresponding Root
	if ex.SelfParent() == root.X {
		spRound = root.Round
		spRoot = true
	} else {
		spRound = h.Round(ex.SelfParent())
		spRoot = false
	}

	opRound := -1
	opRoot := false
	if _, err := h.Store.GetEvent(ex.OtherParent()); err == nil {
		//if we known the other-parent, fetch its Round directly
		opRound = h.Round(ex.OtherParent())
	} else if ex.OtherParent() == root.Y {
		//we do not know the other-parent but it is referenced in Root.Y
		opRound = root.Round
		opRoot = true
	} else if other, ok := root.Others[x]; ok && other == ex.OtherParent() {
		//we do not know the other-parent but it is referenced  in Root.Others
		//we use the Root's Round
		//in reality the OtherParent Round is not necessarily the same as the
		//Root's but it is necessarily smaller. Since We are intererest in the
		//max between self-parent and other-parent rounds, this shortcut is
		//acceptable.
		opRound = root.Round
	}

	res.round = spRound
	res.isRoot = spRoot
	if spRound < opRound {
		res.round = opRound
		res.isRoot = opRoot
	}
	return res
}

//true if x is a witness (first event of a round for the owner)
func (h *Hashgraph) Witness(x string) bool {
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false
	}

	root, err := h.Store.GetRoot(ex.Creator())
	if err != nil {
		return false
	}

	//If it is the creator's first Event, return true
	if ex.SelfParent() == root.X && ex.OtherParent() == root.Y {
		return true
	}

	return h.Round(x) > h.Round(ex.SelfParent())
}

//true if round of x should be incremented
func (h *Hashgraph) RoundInc(x string) bool {

	parentRound := h.ParentRound(x)

	//If parent-round was obtained from a Root, then x is the Event that sits
	//right on top of the Root. RoundInc is true.
	if parentRound.isRoot {
		return true
	}

	//If parent-round was obtained from a regulare Event, then we need to check
	//if x strongly-sees a strong majority of withnesses from parent-round.
	c := 0
	for _, w := range h.Store.RoundWitnesses(parentRound.round) {
		if h.StronglySee(x, w) {
			c++
		}
	}

	return c >= h.SuperMajority()
}

func (h *Hashgraph) RoundReceived(x string) int {

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return -1
	}
	if ex.roundReceived == nil {
		return -1
	}

	return *ex.roundReceived
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

	round := h.ParentRound(x).round

	inc := h.RoundInc(x)

	if inc {
		round++
	}
	return round
}

//round(x) - round(y)
func (h *Hashgraph) RoundDiff(x, y string) (int, error) {

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

func (h *Hashgraph) InsertEvent(event Event, setWireInfo bool) error {
	//verify signature
	if ok, err := event.Verify(); !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("Invalid signature")
	}

	if err := h.CheckSelfParent(event); err != nil {
		return fmt.Errorf("CheckSelfParent: %s", err)
	}

	if err := h.CheckOtherParent(event); err != nil {
		return fmt.Errorf("CheckOtherParent: %s", err)
	}

	event.topologicalIndex = h.topologicalIndex
	h.topologicalIndex++

	if setWireInfo {
		if err := h.SetWireInfo(&event); err != nil {
			return fmt.Errorf("SetWireInfo: %s", err)
		}
	}

	if err := h.InitEventCoordinates(&event); err != nil {
		return fmt.Errorf("InitEventCoordinates: %s", err)
	}

	if err := h.Store.SetEvent(event); err != nil {
		return fmt.Errorf("SetEvent: %s", err)
	}

	if err := h.UpdateAncestorFirstDescendant(event); err != nil {
		return fmt.Errorf("UpdateAncestorFirstDescendant: %s", err)
	}

	h.UndeterminedEvents = append(h.UndeterminedEvents, event.Hex())

	if event.IsLoaded() {
		h.PendingLoadedEvents++
	}

	return nil
}

//Check the SelfParent is the Creator's last known Event
func (h *Hashgraph) CheckSelfParent(event Event) error {
	selfParent := event.SelfParent()
	creator := event.Creator()

	creatorLastKnown, _, err := h.Store.LastFrom(creator)
	if err != nil {
		return err
	}

	selfParentLegit := selfParent == creatorLastKnown

	if !selfParentLegit {
		return fmt.Errorf("Self-parent not last known event by creator")
	}

	return nil
}

//Check if we know the OtherParent
func (h *Hashgraph) CheckOtherParent(event Event) error {
	otherParent := event.OtherParent()
	if otherParent != "" {
		//Check if we have it
		_, err := h.Store.GetEvent(otherParent)
		if err != nil {
			//it might still be in the Root
			root, err := h.Store.GetRoot(event.Creator())
			if err != nil {
				return err
			}
			if root.X == event.SelfParent() && root.Y == otherParent {
				return nil
			}
			other, ok := root.Others[event.Hex()]
			if ok && other == event.OtherParent() {
				return nil
			}
			return fmt.Errorf("Other-parent not known")
		}
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

	selfParent, selfParentError := h.Store.GetEvent(event.SelfParent())
	otherParent, otherParentError := h.Store.GetEvent(event.OtherParent())

	if selfParentError != nil && otherParentError != nil {
		for fakeID := 0; fakeID < members; fakeID++ {
			event.lastAncestors[fakeID] = EventCoordinates{
				index: -1,
			}
		}
	} else if selfParentError != nil {
		copy(event.lastAncestors[:members], otherParent.lastAncestors)
	} else if otherParentError != nil {
		copy(event.lastAncestors[:members], selfParent.lastAncestors)
	} else {
		selfParentLastAncestors := selfParent.lastAncestors
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
				break
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

	//could be the first Event inserted for this creator. In this case, use Root
	if lf, isRoot, _ := h.Store.LastFrom(event.Creator()); isRoot && lf == event.SelfParent() {
		root, err := h.Store.GetRoot(event.Creator())
		if err != nil {
			return err
		}
		selfParentIndex = root.Index
	} else {
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

		if err != nil {
			if !common.Is(err, common.KeyNotFound) {
				return err
			}
			h.UndecidedRounds = append(h.UndecidedRounds, roundNumber)
		}

		roundInfo.AddEvent(hash, witness)
		err = h.Store.SetRound(roundNumber, roundInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

//decide if witnesses are famous
func (h *Hashgraph) DecideFame() error {
	votes := make(map[string](map[string]bool)) //[x][y]=>vote(x,y)

	decidedRounds := map[int]int{} // [round number] => index in h.UndefinedRounds
	defer h.updateUndecidedRounds(decidedRounds)

	for pos, i := range h.UndecidedRounds {
		roundInfo, err := h.Store.GetRound(i)
		if err != nil {
			return err
		}
		for _, x := range roundInfo.Witnesses() {
			if roundInfo.IsDecided(x) {
				continue
			}
		X:
			for j := i + 1; j <= h.Store.LastRound(); j++ {
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
								setVote(votes, y, x, v)
								break X //break out of j loop
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

		//Update decidedRounds and LastConsensusRound if all witnesses have been decided
		if roundInfo.WitnessesDecided() {
			decidedRounds[i] = pos

			if h.LastConsensusRound == nil || i > *h.LastConsensusRound {
				h.setLastConsensusRound(i)
			}
		}

		err = h.Store.SetRound(i, roundInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

//remove items from UndecidedRounds
func (h *Hashgraph) updateUndecidedRounds(decidedRounds map[int]int) {
	newUndecidedRounds := []int{}
	for _, ur := range h.UndecidedRounds {
		if _, ok := decidedRounds[ur]; !ok {
			newUndecidedRounds = append(newUndecidedRounds, ur)
		}
	}
	h.UndecidedRounds = newUndecidedRounds
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
		for i := r + 1; i <= h.Store.LastRound(); i++ {
			tr, err := h.Store.GetRound(i)
			if err != nil && !common.Is(err, common.KeyNotFound) {
				return err
			}

			//skip if some witnesses are left undecided
			if !(tr.WitnessesDecided() && h.UndecidedRounds[0] > i) {
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

func (h *Hashgraph) Reset(roots map[string]Root) error {
	if err := h.Store.Reset(roots); err != nil {
		return err
	}

	h.UndeterminedEvents = []string{}
	h.UndecidedRounds = []int{}
	h.PendingLoadedEvents = 0
	h.topologicalIndex = 0

	cacheSize := h.Store.CacheSize()
	h.ancestorCache = common.NewLRU(cacheSize, nil)
	h.selfAncestorCache = common.NewLRU(cacheSize, nil)
	h.oldestSelfAncestorCache = common.NewLRU(cacheSize, nil)
	h.stronglySeeCache = common.NewLRU(cacheSize, nil)
	h.parentRoundCache = common.NewLRU(cacheSize, nil)
	h.roundCache = common.NewLRU(cacheSize, nil)

	return nil
}

func (h *Hashgraph) GetFrame() (Frame, error) {
	lastConsensusRoundIndex := 0
	if lcr := h.LastConsensusRound; lcr != nil {
		lastConsensusRoundIndex = *lcr
	}

	lastConsensusRound, err := h.Store.GetRound(lastConsensusRoundIndex)
	if err != nil {
		return Frame{}, err
	}

	witnessHashes := lastConsensusRound.Witnesses()

	events := []Event{}
	roots := make(map[string]Root)
	for _, wh := range witnessHashes {
		w, err := h.Store.GetEvent(wh)
		if err != nil {
			return Frame{}, err
		}
		events = append(events, w)
		roots[w.Creator()] = Root{
			X:      w.SelfParent(),
			Y:      w.OtherParent(),
			Index:  w.Index() - 1,
			Round:  h.Round(w.SelfParent()),
			Others: map[string]string{},
		}

		participantEvents, err := h.Store.ParticipantEvents(w.Creator(), w.Index())
		if err != nil {
			return Frame{}, err
		}
		for _, e := range participantEvents {
			ev, err := h.Store.GetEvent(e)
			if err != nil {
				return Frame{}, err
			}
			events = append(events, ev)
		}
	}

	//Not every participant necessarily has a witness in LastConsensusRound.
	//Hence, there could be participants with no Root at this point.
	//For these partcipants, use their last known Event.
	for p := range h.Participants {
		if _, ok := roots[p]; !ok {
			var root Root
			last, isRoot, err := h.Store.LastFrom(p)
			if err != nil {
				return Frame{}, err
			}
			if isRoot {
				root, err = h.Store.GetRoot(p)
				if err != nil {
					return Frame{}, err
				}
			} else {
				ev, err := h.Store.GetEvent(last)
				if err != nil {
					return Frame{}, err
				}
				events = append(events, ev)
				root = Root{
					X:      ev.SelfParent(),
					Y:      ev.OtherParent(),
					Index:  ev.Index() - 1,
					Round:  h.Round(ev.SelfParent()),
					Others: map[string]string{},
				}
			}
			roots[p] = root
		}
	}

	sort.Sort(ByTopologicalOrder(events))

	//Some Events in the Frame might have other-parents that are outside of the
	//Frame (cf root.go ex 2)
	//When inserting these Events in a newly reset hashgraph, the CheckOtherParent
	//method would return an error because the other-parent would not be found.
	//So we make it possible to also look for other-parents in the creator's Root.
	treated := map[string]bool{}
	for _, ev := range events {
		treated[ev.Hex()] = true
		otherParent := ev.OtherParent()
		if otherParent != "" {
			opt, ok := treated[otherParent]
			if !opt || !ok {
				if ev.SelfParent() != roots[ev.Creator()].X {
					roots[ev.Creator()].Others[ev.Hex()] = otherParent
				}
			}
		}
	}

	frame := Frame{
		Roots:  roots,
		Events: events,
	}

	return frame, nil
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
