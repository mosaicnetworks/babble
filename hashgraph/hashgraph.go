package hashgraph

import (
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mosaicnetworks/babble/common"
)

type pendingRound struct {
	Index   int
	Decided bool
}

//Hashgraph is a DAG of Events. It also contains methods to extract a consensus
//order of Events and map them onto a blockchain.
type Hashgraph struct {
	Participants                      map[string]int   //[public key] => id
	ReverseParticipants               map[int]string   //[id] => public key
	Store                             Store            //store of Events, Rounds, and Blocks
	UndeterminedEvents                []string         //[index] => hash . FIFO queue of Events whose consensus order is not yet determined
	PendingRounds                     []*pendingRound  //FIFO queue of Rounds which have not attained consensus yet
	LastConsensusRound                *int             //index of last consensus round
	FirstConsensusRound               *int             //index of first consensus round
	LowestRoundWithUndeterminedEvents *int             //index of lowest round that contains undetermined Events
	AnchorBlock                       *int             //index of last block below LowestRoundWithUndeterminedEvents
	LastCommitedRoundEvents           int              //number of events in round before LastConsensusRound
	SigPool                           []BlockSignature //Pool of Block signatures that need to be processed
	ConsensusTransactions             int              //number of consensus transactions
	PendingLoadedEvents               int              //number of loaded events that are not yet committed
	commitCh                          chan Block       //channel for committing Blocks
	topologicalIndex                  int              //counter used to order events in topological order (only local)
	superMajority                     int
	trustCount                        int

	ancestorCache           *common.LRU
	selfAncestorCache       *common.LRU
	oldestSelfAncestorCache *common.LRU
	stronglySeeCache        *common.LRU
	parentRoundCache        *common.LRU
	roundCache              *common.LRU

	logger *logrus.Entry
}

//NewHashgraph instantiates a Hashgraph from a list of participants, underlying
//data store and commit channel
func NewHashgraph(participants map[string]int, store Store, commitCh chan Block, logger *logrus.Entry) *Hashgraph {
	if logger == nil {
		log := logrus.New()
		log.Level = logrus.DebugLevel
		logger = logrus.NewEntry(log)
	}

	reverseParticipants := make(map[int]string)
	for pk, id := range participants {
		reverseParticipants[id] = pk
	}

	cacheSize := store.CacheSize()
	hashgraph := Hashgraph{
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
		trustCount:              len(participants) / 3,
	}

	return &hashgraph
}

/*******************************************************************************
Private Methods
*******************************************************************************/

//true if y is an ancestor of x
func (h *Hashgraph) ancestor(x, y string) bool {
	if c, ok := h.ancestorCache.Get(Key{x, y}); ok {
		return c.(bool)
	}
	a := h._ancestor(x, y)
	h.ancestorCache.Add(Key{x, y}, a)
	return a
}

func (h *Hashgraph) _ancestor(x, y string) bool {
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
func (h *Hashgraph) selfAncestor(x, y string) bool {
	if c, ok := h.selfAncestorCache.Get(Key{x, y}); ok {
		return c.(bool)
	}
	a := h._selfAncestor(x, y)
	h.selfAncestorCache.Add(Key{x, y}, a)
	return a
}

func (h *Hashgraph) _selfAncestor(x, y string) bool {
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
func (h *Hashgraph) see(x, y string) bool {
	return h.ancestor(x, y)
	//it is not necessary to detect forks because we assume that the InsertEvent
	//function makes it impossible to insert two Events at the same height for
	//the same participant.
}

//oldest self-ancestor of x to see y
func (h *Hashgraph) oldestSelfAncestorToSee(x, y string) string {
	if c, ok := h.oldestSelfAncestorCache.Get(Key{x, y}); ok {
		return c.(string)
	}
	res := h._oldestSelfAncestorToSee(x, y)
	h.oldestSelfAncestorCache.Add(Key{x, y}, res)
	return res
}

func (h *Hashgraph) _oldestSelfAncestorToSee(x, y string) string {
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
func (h *Hashgraph) stronglySee(x, y string) bool {
	if c, ok := h.stronglySeeCache.Get(Key{x, y}); ok {
		return c.(bool)
	}
	ss := h._stronglySee(x, y)
	h.stronglySeeCache.Add(Key{x, y}, ss)
	return ss
}

func (h *Hashgraph) _stronglySee(x, y string) bool {

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
	return c >= h.superMajority
}

//PRI.round: max of parent rounds
//PRI.isRoot: true if round is taken from a Root
func (h *Hashgraph) parentRound(x string) ParentRoundInfo {
	if c, ok := h.parentRoundCache.Get(x); ok {
		return c.(ParentRoundInfo)
	}
	pr := h._parentRound(x)
	h.parentRoundCache.Add(x, pr)
	return pr
}

func (h *Hashgraph) _parentRound(x string) ParentRoundInfo {
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
	spSSWitnesses := 0
	//If it is the creator's first Event, use the corresponding Root
	if ex.SelfParent() == root.X {
		spRound = root.Round
		spRoot = true
		spSSWitnesses = root.StronglySeenWitnesses
	} else {
		spRound = h.round(ex.SelfParent())
		spRoot = false
	}

	opRound := -1
	opRoot := false
	opSSWitnesses := 0
	if _, err := h.Store.GetEvent(ex.OtherParent()); err == nil {
		//if we known the other-parent, fetch its Round directly
		opRound = h.round(ex.OtherParent())
	} else if ex.OtherParent() == root.Y {
		//we do not know the other-parent but it is referenced in Root.Y
		opRound = root.Round
		opRoot = true
		opSSWitnesses = root.StronglySeenWitnesses
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
	res.rootStronglySeenWitnesses = spSSWitnesses
	if spRound < opRound {
		res.round = opRound
		res.isRoot = opRoot
		res.rootStronglySeenWitnesses = opSSWitnesses
	}
	return res
}

//true if x is a witness (first event of a round for the owner)
func (h *Hashgraph) witness(x string) bool {
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

	return h.round(x) > h.round(ex.SelfParent())
}

//true if round of x should be incremented
func (h *Hashgraph) roundInc(x string) bool {

	c := 0

	parentRound := h.parentRound(x)

	if parentRound.isRoot {
		if parentRound.round == -1 {
			return true
		}
		c = parentRound.rootStronglySeenWitnesses
	} else {
		for _, w := range h.Store.RoundWitnesses(parentRound.round) {
			if h.stronglySee(x, w) {
				c++
			}
		}
	}

	return c >= h.superMajority
}

func (h *Hashgraph) roundReceived(x string) int {

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return -1
	}
	if ex.roundReceived == nil {
		return -1
	}

	return *ex.roundReceived
}

func (h *Hashgraph) round(x string) int {
	if c, ok := h.roundCache.Get(x); ok {
		return c.(int)
	}
	r := h._round(x)
	h.roundCache.Add(x, r)
	return r
}

func (h *Hashgraph) _round(x string) int {

	round := h.parentRound(x).round

	inc := h.roundInc(x)

	if inc {
		round++
	}
	return round
}

//round(x) - round(y)
func (h *Hashgraph) roundDiff(x, y string) (int, error) {

	xRound := h.round(x)
	if xRound < 0 {
		return math.MinInt32, fmt.Errorf("event %s has negative round", x)
	}
	yRound := h.round(y)
	if yRound < 0 {
		return math.MinInt32, fmt.Errorf("event %s has negative round", y)
	}

	return xRound - yRound, nil
}

//Check the SelfParent is the Creator's last known Event
func (h *Hashgraph) checkSelfParent(event Event) error {
	selfParent := event.SelfParent()
	creator := event.Creator()

	creatorLastKnown, _, err := h.Store.LastEventFrom(creator)
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
func (h *Hashgraph) checkOtherParent(event Event) error {
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
func (h *Hashgraph) initEventCoordinates(event *Event) error {
	members := len(h.Participants)

	event.firstDescendants = make([]EventCoordinates, members)
	for id := 0; id < members; id++ {
		event.firstDescendants[id] = EventCoordinates{
			index: math.MaxInt32,
		}
	}

	event.lastAncestors = make([]EventCoordinates, members)

	selfParent, selfParentError := h.Store.GetEvent(event.SelfParent())
	otherParent, otherParentError := h.Store.GetEvent(event.OtherParent())

	if selfParentError != nil && otherParentError != nil {
		for id := 0; id < members; id++ {
			event.lastAncestors[id] = EventCoordinates{
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
	creatorID, ok := h.Participants[creator]
	if !ok {
		return fmt.Errorf("Could not find creator id (%s)", creator)
	}
	hash := event.Hex()

	event.firstDescendants[creatorID] = EventCoordinates{index: index, hash: hash}
	event.lastAncestors[creatorID] = EventCoordinates{index: index, hash: hash}

	return nil
}

//update first decendant of each last ancestor to point to event
func (h *Hashgraph) updateAncestorFirstDescendant(event Event) error {
	creatorID, ok := h.Participants[event.Creator()]
	if !ok {
		return fmt.Errorf("Could not find creator id (%s)", event.Creator())
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
			if a.firstDescendants[creatorID].index == math.MaxInt32 {
				a.firstDescendants[creatorID] = EventCoordinates{index: index, hash: hash}
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

func (h *Hashgraph) setWireInfo(event *Event) error {
	selfParentIndex := -1
	otherParentCreatorID := -1
	otherParentIndex := -1

	//could be the first Event inserted for this creator. In this case, use Root
	if lf, isRoot, _ := h.Store.LastEventFrom(event.Creator()); isRoot && lf == event.SelfParent() {
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

func (h *Hashgraph) updatePendingRounds(decidedRounds map[int]int) {
	for _, ur := range h.PendingRounds {
		if _, ok := decidedRounds[ur.Index]; ok {
			ur.Decided = true
		}
	}
}

//Remove processed Signatures from SigPool
func (h *Hashgraph) removeProcessedSignatures(processedSignatures map[int]bool) {
	newSigPool := []BlockSignature{}
	for _, bs := range h.SigPool {
		if _, ok := processedSignatures[bs.Index]; !ok {
			newSigPool = append(newSigPool, bs)
		}
	}
	h.SigPool = newSigPool
}

func (h *Hashgraph) medianTimestamp(eventHashes []string) time.Time {
	events := []Event{}
	for _, x := range eventHashes {
		ex, _ := h.Store.GetEvent(x)
		events = append(events, ex)
	}
	sort.Sort(ByTimestamp(events))

	//If there is an even number of Events, always take the value from upper
	//part of the array. Otherwise, we brake the topological order.
	return events[len(events)/2].Body.Timestamp
}

/*******************************************************************************
Public Methods
*******************************************************************************/

//ReadWireInfo converts a WireEvent to an Event by replacing int IDs with the
//corresponding public keys.
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
		Transactions:    wevent.Body.Transactions,
		BlockSignatures: wevent.BlockSignatures(creatorBytes),
		Parents:         []string{selfParent, otherParent},
		Creator:         creatorBytes,

		Timestamp:            wevent.Body.Timestamp,
		Index:                wevent.Body.Index,
		selfParentIndex:      wevent.Body.SelfParentIndex,
		otherParentCreatorID: wevent.Body.OtherParentCreatorID,
		otherParentIndex:     wevent.Body.OtherParentIndex,
		creatorID:            wevent.Body.CreatorID,
	}

	event := &Event{
		Body:      body,
		Signature: wevent.Signature,
	}

	return event, nil
}

//InsertEvent attempts to insert an Event in the DAG. It verifies the signature,
//checks the ancestors are known, and prevents the introduction of forks.
func (h *Hashgraph) InsertEvent(event Event, setWireInfo bool) error {
	//verify signature
	if ok, err := event.Verify(); !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("Invalid Event signature")
	}

	if err := h.checkSelfParent(event); err != nil {
		return fmt.Errorf("CheckSelfParent: %s", err)
	}

	if err := h.checkOtherParent(event); err != nil {
		return fmt.Errorf("CheckOtherParent: %s", err)
	}

	event.topologicalIndex = h.topologicalIndex
	h.topologicalIndex++

	if setWireInfo {
		if err := h.setWireInfo(&event); err != nil {
			return fmt.Errorf("SetWireInfo: %s", err)
		}
	}

	if err := h.initEventCoordinates(&event); err != nil {
		return fmt.Errorf("InitEventCoordinates: %s", err)
	}

	if err := h.Store.SetEvent(event); err != nil {
		return fmt.Errorf("SetEvent: %s", err)
	}

	if err := h.updateAncestorFirstDescendant(event); err != nil {
		return fmt.Errorf("UpdateAncestorFirstDescendant: %s", err)
	}

	h.UndeterminedEvents = append(h.UndeterminedEvents, event.Hex())

	if event.IsLoaded() {
		h.PendingLoadedEvents++
	}

	h.SigPool = append(h.SigPool, event.BlockSignatures()...)

	return nil
}

//DivideRounds assigns a Round to Events and flags them as witnesses if
//necessary.
//XXX we could optimize here _ this function is called many times on the same
//events until they are moved out of the UndeterminedEvents queue
func (h *Hashgraph) DivideRounds() error {

	for _, hash := range h.UndeterminedEvents {

		roundNumber := h.round(hash)

		roundInfo, err := h.Store.GetRound(roundNumber)
		if err != nil && !common.Is(err, common.KeyNotFound) {
			return err
		}

		/*
			Why the lower bound?
			Normally, once a Round has attained consensus, it is impossible for
			new Events from a previous Round to be inserted; the lower bound
			appears redundant. This is the case when the hashgraph grows
			linearly, without jumps, which is what we intend by 'Normally'.
			But the Reset function introduces a dicontinuity  by jumping
			straight to a specific place in the hashgraph. This technique relies
			on a base layer of Events (the corresponding Frame's Events) for
			other Events to be added on top, but the base layer must not be
			reprocessed.
		*/
		if !roundInfo.queued &&
			(h.LastConsensusRound == nil ||
				roundNumber >= *h.LastConsensusRound) {

			h.PendingRounds = append(h.PendingRounds, &pendingRound{roundNumber, false})
			roundInfo.queued = true
		}

		witness := h.witness(hash)
		roundInfo.AddEvent(hash, witness)

		err = h.Store.SetRound(roundNumber, roundInfo)
		if err != nil {
			return err
		}
	}

	if l := len(h.PendingRounds); l > 0 {
		h.setLowestRoundWithUndeterminedEvents(h.PendingRounds[l-1].Index)
	}

	return nil
}

//DecideFame decides if witnesses are famous
func (h *Hashgraph) DecideFame() error {

	//XXX is the compiler Smart?
	votes := make(map[string](map[string]bool)) //[x][y]=>vote(x,y)
	setVote := func(votes map[string]map[string]bool, x, y string, vote bool) {
		if votes[x] == nil {
			votes[x] = make(map[string]bool)
		}
		votes[x][y] = vote
	}

	decidedRounds := map[int]int{} // [round number] => index in h.PendingRounds

	for pos, r := range h.PendingRounds {
		roundIndex := r.Index
		roundInfo, err := h.Store.GetRound(roundIndex)
		if err != nil {
			return err
		}
		for _, x := range roundInfo.Witnesses() {
			if roundInfo.IsDecided(x) {
				continue
			}
		VOTE_LOOP:
			for j := roundIndex + 1; j <= h.Store.LastRound(); j++ {
				for _, y := range h.Store.RoundWitnesses(j) {
					diff := j - roundIndex
					if diff == 1 {
						setVote(votes, y, x, h.see(y, x))
					} else {
						//count votes
						ssWitnesses := []string{}
						for _, w := range h.Store.RoundWitnesses(j - 1) {
							if h.stronglySee(y, w) {
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
							if t >= h.superMajority {
								roundInfo.SetFame(x, v)
								setVote(votes, y, x, v)
								break VOTE_LOOP //break out of j loop
							} else {
								setVote(votes, y, x, v)
							}
						} else { //coin round
							if t >= h.superMajority {
								setVote(votes, y, x, v)
							} else {
								setVote(votes, y, x, middleBit(y)) //middle bit of y's hash
							}
						}
					}
				}
			}
		}

		err = h.Store.SetRound(roundIndex, roundInfo)
		if err != nil {
			return err
		}

		if roundInfo.WitnessesDecided() {
			decidedRounds[roundIndex] = pos
		}

	}

	h.updatePendingRounds(decidedRounds)
	return nil
}

//DecideRoundReceived assigns a RoundReceived and timestamp to undetermined
//events when they reach consensus
func (h *Hashgraph) DecideRoundReceived() error {

	newUndeterminedEvents := []string{}

	/* From whitepaper - 18/03/18
	   "[...] An event is said to be “received” in the first round where all the
	   unique famous witnesses have received it, if all earlier rounds have the
	   fame of all witnesses decided"
	*/
	for _, x := range h.UndeterminedEvents {

		received := false
		r := h.round(x)

		for i := r + 1; i <= h.Store.LastRound(); i++ {

			tr, err := h.Store.GetRound(i)
			if err != nil {
				return err
			}

			//We are looping from earlier to later rounds; so if we encounter
			//one round with undecided witnesses, we are sure that this event
			//is not "received". Break out of i loop
			if !(tr.WitnessesDecided()) {
				break
			}

			fws := tr.FamousWitnesses()
			//set of famous witnesses that see x
			s := []string{}
			for _, w := range fws {
				if h.see(w, x) {
					s = append(s, w)
				}
			}

			if len(s) == len(fws) && len(s) > 0 {

				received = true

				ex, err := h.Store.GetEvent(x)
				if err != nil {
					return err
				}
				ex.SetRoundReceived(i)

				t := []string{}
				for _, a := range s {
					t = append(t, h.oldestSelfAncestorToSee(a, x))
				}

				ex.consensusTimestamp = h.medianTimestamp(t)

				err = h.Store.SetEvent(ex)
				if err != nil {
					return err
				}

				tr.SetConsensusEvent(x)
				err = h.Store.SetRound(i, tr)
				if err != nil {
					return err
				}

				//break out of i loop
				break
			}

		}
		if !received {
			newUndeterminedEvents = append(newUndeterminedEvents, x)
			if r := h.round(x); h.LowestRoundWithUndeterminedEvents == nil ||
				r < *h.LowestRoundWithUndeterminedEvents {
				h.setLowestRoundWithUndeterminedEvents(r)
			}
		}
	}

	h.UndeterminedEvents = newUndeterminedEvents

	return nil
}

//ProcessDecidedRounds takes Rounds whose witnesses are decided, computes the
//corresponding Frames, maps them into Blocks, and commits the Blocks via the
//commit channel
func (h *Hashgraph) ProcessDecidedRounds() error {

	//Defer removing processed Rounds from the PendingRounds Queue
	processedIndex := 0
	defer func() {
		h.PendingRounds = h.PendingRounds[processedIndex:]
	}()

	for _, r := range h.PendingRounds {

		//Although it is possible for a Round to be 'decided' before a previous
		//round, we should NEVER process a decided round before all the previous
		//rounds are processed.
		if !r.Decided {
			break
		}

		//This is similar to the lower bound introduced in DivideRounds; it is
		//redundant in normal operations, but becomes necessary after a Reset.
		//Indeed, after a Reset, LastConsensusRound is added to PendingRounds,
		//but its ConsensusEvents (which are necessarily 'under' this Round) are
		//already deemed committed. Hence, skip this Round after a Reset.
		if h.LastConsensusRound != nil && r.Index == *h.LastConsensusRound {
			continue
		}

		frame, err := h.GetFrame(r.Index)
		if err != nil {
			return fmt.Errorf("Getting Frame %d: %v", r.Index, err)
		}

		if len(frame.Events) > 0 {

			//XXX This is ugly. Should be somewhere else; or nowhere
			for _, e := range frame.Events {
				err := h.Store.AddConsensusEvent(e)
				if err != nil {
					return err
				}
				h.ConsensusTransactions += len(e.Transactions())
				if e.IsLoaded() {
					h.PendingLoadedEvents--
				}
			}

			lastBlockIndex := h.Store.LastBlockIndex()
			block, err := NewBlockFromFrame(lastBlockIndex+1, frame)
			if err != nil {
				return err
			}
			if err := h.Store.SetBlock(block); err != nil {
				return err
			}

			if h.commitCh != nil {
				h.commitCh <- block
			}

		} else {
			h.logger.Debugf("No Events to commit for ConsensusRound %d", r.Index)
		}

		processedIndex++

		if h.LastConsensusRound == nil || r.Index > *h.LastConsensusRound {
			h.setLastConsensusRound(r.Index)
		}

	}

	return nil
}

//GetFrame computes the Frame corresponding to a RoundReceived.
func (h *Hashgraph) GetFrame(roundReceived int) (Frame, error) {

	//Try to get it from the Store first
	frame, err := h.Store.GetFrame(roundReceived)
	if err == nil || !common.Is(err, common.KeyNotFound) {
		return frame, err
	}

	//Get the Round and corresponding consensus Events
	round, err := h.Store.GetRound(roundReceived)
	if err != nil {
		return Frame{}, err
	}

	events := []Event{}
	roots := make(map[string]Root)
	for _, eh := range round.ConsensusEvents() {
		e, err := h.Store.GetEvent(eh)
		if err != nil {
			return Frame{}, err
		}
		events = append(events, e)
	}

	//Compute the partial topological order. This is necessary for knowing which
	//events need a Root
	sort.Sort(ByTopologicalOrder(events))

	// Get/Create Roots

	//The events are in topological order. Each time we run into the first Event
	//of a participant, we create a Root for it.
	for _, ev := range events {
		p := ev.Creator()
		if _, ok := roots[p]; !ok {
			//XXX
			parentRound := h.parentRound(ev.Hex())
			c := 0
			for _, w := range h.Store.RoundWitnesses(parentRound.round) {
				if h.stronglySee(ev.Hex(), w) {
					c++
				}
			}
			roots[ev.Creator()] = Root{
				X:     ev.SelfParent(),
				Y:     ev.OtherParent(),
				Index: ev.Index() - 1,
				Round: parentRound.round,
				StronglySeenWitnesses: c,
				Others:                map[string]string{},
			}
		}
	}

	//Every participant needs a Root in the Frame. For the participants that
	//have no Events in this Frame, we just get their latest known Root.
	for p, id := range h.Participants {
		if _, ok := roots[p]; !ok {
			var root Root

			//Get latest known root before roundReceived by recursion
			if h.FirstConsensusRound != nil && roundReceived > *h.FirstConsensusRound {
				prevFrame, err := h.GetFrame(roundReceived - 1)
				if err != nil {
					return Frame{}, fmt.Errorf("GetFrame recursion: %v", err)
				}
				root = prevFrame.Roots[id]
			} else {
				//XXX
				disp := "nil"
				if h.FirstConsensusRound != nil {
					disp = strconv.Itoa(*h.FirstConsensusRound)
				}
				h.logger.WithFields(logrus.Fields{
					"round_received":        roundReceived,
					"first_consensus_round": disp,
				}).Debug("GetFrame recursion")
				lowestRoot, err := h.Store.GetRoot(p)
				if err != nil {
					return Frame{}, err
				}
				root = lowestRoot
			}

			roots[p] = root
		}
	}

	//order roots
	orderedRoots := make([]Root, len(h.Participants))
	for i := 0; i < len(h.Participants); i++ {
		orderedRoots[i] = roots[h.ReverseParticipants[i]]
	}

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

	//Compute the 'fair' order of Events; all the nodes should compute the same
	//order
	sort.Sort(ByConsensusTimestamp(events))

	res := Frame{
		Round:  roundReceived,
		Roots:  orderedRoots,
		Events: events,
	}

	if err := h.Store.SetFrame(res); err != nil {
		return Frame{}, err
	}

	return res, nil
}

//ProcessSigPool runs through the SignaturePool and tries to map a Signature to
//a known Block. If a Signature is found to be valid for a known Block, it is
//appended to the block and removed from the SignaturePool
func (h *Hashgraph) ProcessSigPool() error {
	processedSignatures := map[int]bool{} //index in SigPool => Processed?
	defer h.removeProcessedSignatures(processedSignatures)

	for i, bs := range h.SigPool {
		//check if validator belongs to list of participants
		validatorHex := fmt.Sprintf("0x%X", bs.Validator)
		if _, ok := h.Participants[validatorHex]; !ok {
			h.logger.WithFields(logrus.Fields{
				"index":     bs.Index,
				"validator": validatorHex,
			}).Warning("Verifying Block signature. Unknown validator")
			continue
		}

		block, err := h.Store.GetBlock(bs.Index)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"index": bs.Index,
				"msg":   err,
			}).Warning("Verifying Block signature. Could not fetch Block")
			continue
		}
		valid, err := block.Verify(bs)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"index": bs.Index,
				"msg":   err,
			}).Error("Verifying Block signature")
			return err
		}
		if !valid {
			h.logger.WithFields(logrus.Fields{
				"index":     bs.Index,
				"validator": h.Participants[validatorHex],
				"block":     block,
			}).Warning("Verifying Block signature. Invalid signature")
			continue
		}

		block.SetSignature(bs)

		if err := h.Store.SetBlock(block); err != nil {
			h.logger.WithFields(logrus.Fields{
				"index": bs.Index,
				"msg":   err,
			}).Warning("Saving Block")
		}

		if len(block.Signatures) > h.trustCount &&
			(h.AnchorBlock == nil ||
				block.Index() > *h.AnchorBlock) &&
			(h.LowestRoundWithUndeterminedEvents == nil ||
				block.RoundReceived() < *h.LowestRoundWithUndeterminedEvents) {
			h.setAnchorBlock(block.Index())
			h.logger.WithFields(logrus.Fields{
				"block_index": block.Index(),
				"signatures":  len(block.Signatures),
				"trustCount":  h.trustCount,
			}).Debug("Setting AnchorBlock")
		}

		processedSignatures[i] = true
	}

	return nil
}

//GetLatestFrame returns the Frame corresponding to the last consensus round.
//It is not safe to Reset a hashgraph from this Frame because it does not
//necessarily correspond to an Anchor Block (there could be undetermined Events
//below the Frame.
func (h *Hashgraph) GetLatestFrame() (Frame, error) {
	lastConsensusRoundIndex := 0
	if lcr := h.LastConsensusRound; lcr != nil {
		lastConsensusRoundIndex = *lcr
	}

	return h.GetFrame(lastConsensusRoundIndex)
}

//GetAnchorBlockWithFrame returns the AnchorBlock and the corresponding Frame.
//This can be used as a base to Reset a Hashgraph
func (h *Hashgraph) GetAnchorBlockWithFrame() (Block, Frame, error) {

	if h.AnchorBlock == nil {
		return Block{}, Frame{}, fmt.Errorf("No Anchor Block")
	}

	block, err := h.Store.GetBlock(*h.AnchorBlock)
	if err != nil {
		return Block{}, Frame{}, err
	}

	frame, err := h.GetFrame(block.RoundReceived())
	if err != nil {
		return Block{}, Frame{}, err
	}

	return block, frame, nil
}

//Reset clears the Hashgraph and resets it from a new base.
func (h *Hashgraph) Reset(block Block, frame Frame) error {

	//Clear all state
	h.LastConsensusRound = nil
	h.FirstConsensusRound = nil
	h.AnchorBlock = nil
	h.LowestRoundWithUndeterminedEvents = nil

	h.UndeterminedEvents = []string{}
	h.PendingRounds = []*pendingRound{}
	h.PendingLoadedEvents = 0
	h.topologicalIndex = 0

	cacheSize := h.Store.CacheSize()
	h.ancestorCache = common.NewLRU(cacheSize, nil)
	h.selfAncestorCache = common.NewLRU(cacheSize, nil)
	h.oldestSelfAncestorCache = common.NewLRU(cacheSize, nil)
	h.stronglySeeCache = common.NewLRU(cacheSize, nil)
	h.parentRoundCache = common.NewLRU(cacheSize, nil)
	h.roundCache = common.NewLRU(cacheSize, nil)

	//Initialize new Roots
	rootMap := map[string]Root{}
	for id, root := range frame.Roots {
		p, ok := h.ReverseParticipants[id]
		if !ok {
			return fmt.Errorf("Could not find participant with id %d", id)
		}
		rootMap[p] = root
	}
	if err := h.Store.Reset(rootMap); err != nil {
		return err
	}

	//Insert Block
	if err := h.Store.SetBlock(block); err != nil {
		return err
	}

	h.setLastConsensusRound(block.RoundReceived())

	//XXX
	sort.Sort(ByTopologicalOrder(frame.Events))
	//Insert Frame Events
	for _, ev := range frame.Events {
		if err := h.InsertEvent(ev, false); err != nil {
			return err
		}
	}

	return nil
}

//Bootstrap loads all Events from the Store's DB (if there is one) and feeds
//them to the Hashgraph (in topological order) for consensus ordering. After this
//method call, the Hashgraph should be in a state coherent with the 'tip' of the
//Hashgraph
func (h *Hashgraph) Bootstrap() error {
	if badgerStore, ok := h.Store.(*BadgerStore); ok {
		//Retreive the Events from the underlying DB. They come out in topological
		//order
		topologicalEvents, err := badgerStore.dbTopologicalEvents()
		if err != nil {
			return err
		}

		//Insert the Events in the Hashgraph
		for _, e := range topologicalEvents {
			if err := h.InsertEvent(e, true); err != nil {
				return err
			}
		}

		//Compute the consensus order of Events
		if err := h.DivideRounds(); err != nil {
			return err
		}
		if err := h.DecideFame(); err != nil {
			return err
		}
		if err := h.DecideRoundReceived(); err != nil {
			return err
		}
		if err := h.ProcessDecidedRounds(); err != nil {
			return err
		}
		if err := h.ProcessSigPool(); err != nil {
			return err
		}
	}

	return nil
}

/*******************************************************************************
Setters
*******************************************************************************/

func (h *Hashgraph) setLastConsensusRound(i int) {
	if h.LastConsensusRound == nil {
		h.LastConsensusRound = new(int)
	}
	*h.LastConsensusRound = i

	if h.FirstConsensusRound == nil {
		h.FirstConsensusRound = new(int)
		*h.FirstConsensusRound = i
	}
}

func (h *Hashgraph) setLowestRoundWithUndeterminedEvents(index int) {
	if h.LowestRoundWithUndeterminedEvents == nil {
		h.LowestRoundWithUndeterminedEvents = new(int)
	}
	*h.LowestRoundWithUndeterminedEvents = index
}

func (h *Hashgraph) setAnchorBlock(i int) {
	if h.AnchorBlock == nil {
		h.AnchorBlock = new(int)
	}
	*h.AnchorBlock = i
}

/*******************************************************************************
   Helpers
*******************************************************************************/

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
