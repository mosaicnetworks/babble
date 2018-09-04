package hashgraph

import (
	"encoding/hex"
	"fmt"
	"math"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/mosaicnetworks/babble/common"
)

//Hashgraph is a DAG of Events. It also contains methods to extract a consensus
//order of Events and map them onto a blockchain.
type Hashgraph struct {
	Participants            map[string]int   //[public key] => id
	ReverseParticipants     map[int]string   //[id] => public key
	Store                   Store            //store of Events, Rounds, and Blocks
	UndeterminedEvents      []string         //[index] => hash . FIFO queue of Events whose consensus order is not yet determined
	PendingRounds           []*pendingRound  //FIFO queue of Rounds which have not attained consensus yet
	LastConsensusRound      *int             //index of last consensus round
	FirstConsensusRound     *int             //index of first consensus round (only used in tests)
	AnchorBlock             *int             //index of last block with enough signatures
	LastCommitedRoundEvents int              //number of events in round before LastConsensusRound
	SigPool                 []BlockSignature //Pool of Block signatures that need to be processed
	ConsensusTransactions   int              //number of consensus transactions
	PendingLoadedEvents     int              //number of loaded events that are not yet committed
	commitCh                chan Block       //channel for committing Blocks
	topologicalIndex        int              //counter used to order events in topological order (only local)
	superMajority           int
	trustCount              int

	ancestorCache     *common.LRU
	selfAncestorCache *common.LRU
	stronglySeeCache  *common.LRU
	roundCache        *common.LRU
	timestampCache    *common.LRU

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

	superMajority := 2*len(participants)/3 + 1
	trustCount := int(math.Ceil(float64(len(participants)) / float64(3)))

	cacheSize := store.CacheSize()
	hashgraph := Hashgraph{
		Participants:        participants,
		ReverseParticipants: reverseParticipants,
		Store:               store,
		commitCh:            commitCh,
		ancestorCache:       common.NewLRU(cacheSize, nil),
		selfAncestorCache:   common.NewLRU(cacheSize, nil),
		stronglySeeCache:    common.NewLRU(cacheSize, nil),
		roundCache:          common.NewLRU(cacheSize, nil),
		timestampCache:      common.NewLRU(cacheSize, nil),
		logger:              logger,
		superMajority:       superMajority,
		trustCount:          trustCount,
	}

	return &hashgraph
}

/*******************************************************************************
Private Methods
*******************************************************************************/

//true if y is an ancestor of x
func (h *Hashgraph) ancestor(x, y string) (bool, error) {
	if c, ok := h.ancestorCache.Get(Key{x, y}); ok {
		return c.(bool), nil
	}
	a, err := h._ancestor(x, y)
	if err != nil {
		return false, err
	}
	h.ancestorCache.Add(Key{x, y}, a)
	return a, nil
}

func (h *Hashgraph) _ancestor(x, y string) (bool, error) {

	if x == y {
		return true, nil
	}

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false, err
	}

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false, err
	}

	eyCreator := h.Participants[ey.Creator()]
	lastAncestorKnownFromYCreator := ex.lastAncestors[eyCreator].index

	return lastAncestorKnownFromYCreator >= ey.Index(), nil
}

//true if y is a self-ancestor of x
func (h *Hashgraph) selfAncestor(x, y string) (bool, error) {
	if c, ok := h.selfAncestorCache.Get(Key{x, y}); ok {
		return c.(bool), nil
	}
	a, err := h._selfAncestor(x, y)
	if err != nil {
		return false, err
	}
	h.selfAncestorCache.Add(Key{x, y}, a)
	return a, nil
}

func (h *Hashgraph) _selfAncestor(x, y string) (bool, error) {
	if x == y {
		return true, nil
	}
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false, err
	}
	exCreator := h.Participants[ex.Creator()]

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false, err
	}
	eyCreator := h.Participants[ey.Creator()]

	return exCreator == eyCreator && ex.Index() >= ey.Index(), nil
}

//true if x sees y
func (h *Hashgraph) see(x, y string) (bool, error) {
	return h.ancestor(x, y)
	//it is not necessary to detect forks because we assume that the InsertEvent
	//function makes it impossible to insert two Events at the same height for
	//the same participant.
}

//true if x strongly sees y
func (h *Hashgraph) stronglySee(x, y string) (bool, error) {
	if c, ok := h.stronglySeeCache.Get(Key{x, y}); ok {
		return c.(bool), nil
	}
	ss, err := h._stronglySee(x, y)
	if err != nil {
		return false, err
	}
	h.stronglySeeCache.Add(Key{x, y}, ss)
	return ss, nil
}

func (h *Hashgraph) _stronglySee(x, y string) (bool, error) {

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false, err
	}

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false, err
	}

	c := 0
	for i := 0; i < len(ex.lastAncestors); i++ {
		if ex.lastAncestors[i].index >= ey.firstDescendants[i].index {
			c++
		}
	}
	return c >= h.superMajority, nil
}

func (h *Hashgraph) round(x string) (int, error) {
	if c, ok := h.roundCache.Get(x); ok {
		return c.(int), nil
	}
	r, err := h._round(x)
	if err != nil {
		return -1, err
	}
	h.roundCache.Add(x, r)
	return r, nil
}

func (h *Hashgraph) _round(x string) (int, error) {

	/*
		x is the Root
		Use Root.SelfParent.Round
	*/
	rootsBySelfParent, _ := h.Store.RootsBySelfParent()
	if r, ok := rootsBySelfParent[x]; ok {
		return r.SelfParent.Round, nil
	}

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return math.MinInt32, err
	}

	root, err := h.Store.GetRoot(ex.Creator())
	if err != nil {
		return math.MinInt32, err
	}

	/*
		The Event is directly attached to the Root.
	*/
	if ex.SelfParent() == root.SelfParent.Hash {
		//Root is authoritative EXCEPT if other-parent is not in the root
		if other, ok := root.Others[ex.Hex()]; (ex.OtherParent() == "") ||
			(ok && other.Hash == ex.OtherParent()) {

			return root.NextRound, nil
		}
	}

	/*
		The Event's parents are "normal" Events.
		Use the whitepaper formula: parentRound + roundInc
	*/
	parentRound, err := h.round(ex.SelfParent())
	if err != nil {
		return math.MinInt32, err
	}
	if ex.OtherParent() != "" {
		var opRound int
		//XXX
		if other, ok := root.Others[ex.Hex()]; ok && other.Hash == ex.OtherParent() {
			opRound = root.NextRound
		} else {
			opRound, err = h.round(ex.OtherParent())
			if err != nil {
				return math.MinInt32, err
			}
		}

		if opRound > parentRound {
			parentRound = opRound
		}
	}

	c := 0
	for _, w := range h.Store.RoundWitnesses(parentRound) {
		ss, err := h.stronglySee(x, w)
		if err != nil {
			return math.MinInt32, err
		}
		if ss {
			c++
		}
	}
	if c >= h.superMajority {
		parentRound++
	}

	return parentRound, nil
}

//true if x is a witness (first event of a round for the owner)
func (h *Hashgraph) witness(x string) (bool, error) {
	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false, err
	}

	xRound, err := h.round(x)
	if err != nil {
		return false, err
	}
	spRound, err := h.round(ex.SelfParent())
	if err != nil {
		return false, err
	}
	return xRound > spRound, nil
}

func (h *Hashgraph) roundReceived(x string) (int, error) {

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return -1, err
	}

	res := -1
	if ex.roundReceived != nil {
		res = *ex.roundReceived
	}

	return res, nil
}

func (h *Hashgraph) lamportTimestamp(x string) (int, error) {
	if c, ok := h.timestampCache.Get(x); ok {
		return c.(int), nil
	}
	r, err := h._lamportTimestamp(x)
	if err != nil {
		return -1, err
	}
	h.timestampCache.Add(x, r)
	return r, nil
}

func (h *Hashgraph) _lamportTimestamp(x string) (int, error) {
	/*
		x is the Root
		User Root.SelfParent.LamportTimestamp
	*/
	rootsBySelfParent, _ := h.Store.RootsBySelfParent()
	if r, ok := rootsBySelfParent[x]; ok {
		return r.SelfParent.LamportTimestamp, nil
	}

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return math.MinInt32, err
	}

	//We are going to need the Root later
	root, err := h.Store.GetRoot(ex.Creator())
	if err != nil {
		return math.MinInt32, err
	}

	plt := math.MinInt32
	//If it is the creator's first Event, use the corresponding Root
	if ex.SelfParent() == root.SelfParent.Hash {
		plt = root.SelfParent.LamportTimestamp
	} else {
		t, err := h.lamportTimestamp(ex.SelfParent())
		if err != nil {
			return math.MinInt32, err
		}
		plt = t
	}

	if ex.OtherParent() != "" {
		opLT := math.MinInt32
		if _, err := h.Store.GetEvent(ex.OtherParent()); err == nil {
			//if we know the other-parent, fetch its Round directly
			t, err := h.lamportTimestamp(ex.OtherParent())
			if err != nil {
				return math.MinInt32, err
			}
			opLT = t
		} else if other, ok := root.Others[x]; ok && other.Hash == ex.OtherParent() {
			//we do not know the other-parent but it is referenced  in Root.Others
			//we use the Root's LamportTimestamp
			opLT = other.LamportTimestamp
		}

		if opLT > plt {
			plt = opLT
		}
	}

	return plt + 1, nil
}

//round(x) - round(y)
func (h *Hashgraph) roundDiff(x, y string) (int, error) {

	xRound, err := h.round(x)
	if err != nil {
		return math.MinInt32, fmt.Errorf("event %s has negative round", x)
	}

	yRound, err := h.round(y)
	if err != nil {
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
			other, ok := root.Others[event.Hex()]
			if ok && other.Hash == event.OtherParent() {
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

func (h *Hashgraph) createSelfParentRootEvent(ev Event) (RootEvent, error) {
	sp := ev.SelfParent()
	spLT, err := h.lamportTimestamp(sp)
	if err != nil {
		return RootEvent{}, err
	}
	spRound, err := h.round(sp)
	if err != nil {
		return RootEvent{}, err
	}
	selfParentRootEvent := RootEvent{
		Hash:             sp,
		CreatorID:        h.Participants[ev.Creator()],
		Index:            ev.Index() - 1,
		LamportTimestamp: spLT,
		Round:            spRound,
	}
	return selfParentRootEvent, nil
}

func (h *Hashgraph) createOtherParentRootEvent(ev Event) (RootEvent, error) {

	op := ev.OtherParent()

	//it might still be in the Root
	root, err := h.Store.GetRoot(ev.Creator())
	if err != nil {
		return RootEvent{}, err
	}
	if other, ok := root.Others[ev.Hex()]; ok && other.Hash == op {
		return other, nil
	}

	otherParent, err := h.Store.GetEvent(op)
	if err != nil {
		return RootEvent{}, err
	}
	opLT, err := h.lamportTimestamp(op)
	if err != nil {
		return RootEvent{}, err
	}
	opRound, err := h.round(op)
	if err != nil {
		return RootEvent{}, err
	}
	otherParentRootEvent := RootEvent{
		Hash:             op,
		CreatorID:        h.Participants[otherParent.Creator()],
		Index:            otherParent.Index(),
		LamportTimestamp: opLT,
		Round:            opRound,
	}
	return otherParentRootEvent, nil

}

func (h *Hashgraph) createRoot(ev Event) (Root, error) {

	evRound, err := h.round(ev.Hex())
	if err != nil {
		return Root{}, err
	}

	/*
		SelfParent
	*/
	selfParentRootEvent, err := h.createSelfParentRootEvent(ev)
	if err != nil {
		return Root{}, err
	}

	/*
		OtherParent
	*/
	var otherParentRootEvent *RootEvent
	if ev.OtherParent() != "" {
		opre, err := h.createOtherParentRootEvent(ev)
		if err != nil {
			return Root{}, err
		}
		otherParentRootEvent = &opre
	}

	root := Root{
		NextRound:  evRound,
		SelfParent: selfParentRootEvent,
		Others:     map[string]RootEvent{},
	}

	if otherParentRootEvent != nil {
		root.Others[ev.Hex()] = *otherParentRootEvent
	}

	return root, nil
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
		selfParentIndex = root.SelfParent.Index
	} else {
		selfParent, err := h.Store.GetEvent(event.SelfParent())
		if err != nil {
			return err
		}
		selfParentIndex = selfParent.Index()
	}

	if event.OtherParent() != "" {
		//Check Root then regular Events
		root, err := h.Store.GetRoot(event.Creator())
		if err != nil {
			return err
		}
		if other, ok := root.Others[event.Hex()]; ok && other.Hash == event.OtherParent() {
			otherParentCreatorID = other.CreatorID
			otherParentIndex = other.Index
		} else {
			otherParent, err := h.Store.GetEvent(event.OtherParent())
			if err != nil {
				return err
			}
			otherParentCreatorID = h.Participants[otherParent.Creator()]
			otherParentIndex = otherParent.Index()
		}
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

/*******************************************************************************
Public Methods
*******************************************************************************/

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

/*
DivideRounds assigns a Round and LamportTimestamp to Events, and flags them as
witnesses if necessary. Pushes Rounds in the PendingRounds queue if necessary.
*/
func (h *Hashgraph) DivideRounds() error {

	for _, hash := range h.UndeterminedEvents {

		ev, err := h.Store.GetEvent(hash)
		if err != nil {
			return err
		}

		updateEvent := false

		/*
		   Compute Event's round, update the corresponding Round object, and
		   add it to the PendingRounds queue if necessary.
		*/
		if ev.round == nil {

			roundNumber, err := h.round(hash)
			if err != nil {
				return err
			}

			ev.SetRound(roundNumber)
			updateEvent = true

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

			witness, err := h.witness(hash)
			if err != nil {
				return err
			}
			roundInfo.AddEvent(hash, witness)

			err = h.Store.SetRound(roundNumber, roundInfo)
			if err != nil {
				return err
			}
		}

		/*
			Compute the Event's LamportTimestamp
		*/
		if ev.lamportTimestamp == nil {

			lamportTimestamp, err := h.lamportTimestamp(hash)
			if err != nil {
				return err
			}

			ev.SetLamportTimestamp(lamportTimestamp)
			updateEvent = true
		}

		if updateEvent {
			h.Store.SetEvent(ev)
		}
	}

	return nil
}

//DecideFame decides if witnesses are famous
func (h *Hashgraph) DecideFame() error {

	//Initialize the vote map
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
						ycx, err := h.see(y, x)
						if err != nil {
							return err
						}
						setVote(votes, y, x, ycx)
					} else {
						//count votes
						ssWitnesses := []string{}
						for _, w := range h.Store.RoundWitnesses(j - 1) {
							ss, err := h.stronglySee(y, w)
							if err != nil {
								return err
							}
							if ss {
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

//DecideRoundReceived assigns a RoundReceived to undetermined events when they
//reach consensus
func (h *Hashgraph) DecideRoundReceived() error {

	newUndeterminedEvents := []string{}

	/* From whitepaper - 18/03/18
	   "[...] An event is said to be “received” in the first round where all the
	   unique famous witnesses have received it, if all earlier rounds have the
	   fame of all witnesses decided"
	*/
	for _, x := range h.UndeterminedEvents {

		received := false
		r, err := h.round(x)
		if err != nil {
			return err
		}

		for i := r + 1; i <= h.Store.LastRound(); i++ {

			tr, err := h.Store.GetRound(i)
			if err != nil {
				//Can happen after a Reset/FastSync
				if h.LastConsensusRound != nil &&
					r < *h.LastConsensusRound {
					received = true
					break
				}
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
				see, err := h.see(w, x)
				if err != nil {
					return err
				}
				if see {
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

		round, err := h.Store.GetRound(r.Index)
		if err != nil {
			return err
		}
		h.logger.WithFields(logrus.Fields{
			"round_received": r.Index,
			"witnesses":      round.FamousWitnesses(),
			"events":         len(frame.Events),
			"roots":          frame.Roots,
		}).Debugf("Processing Decided Round")

		if len(frame.Events) > 0 {

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
	for _, eh := range round.ConsensusEvents() {
		e, err := h.Store.GetEvent(eh)
		if err != nil {
			return Frame{}, err
		}
		events = append(events, e)
	}

	sort.Sort(ByLamportTimestamp(events))

	// Get/Create Roots
	roots := make(map[string]Root)
	//The events are in topological order. Each time we run into the first Event
	//of a participant, we create a Root for it.
	for _, ev := range events {
		p := ev.Creator()
		if _, ok := roots[p]; !ok {
			root, err := h.createRoot(ev)
			if err != nil {
				return Frame{}, err
			}
			roots[ev.Creator()] = root
		}
	}

	//Every participant needs a Root in the Frame. For the participants that
	//have no Events in this Frame, we create a Root from their last consensus
	//Event, or their last known Root
	for p := range h.Participants {
		if _, ok := roots[p]; !ok {
			var root Root
			lastConsensusEventHash, isRoot, err := h.Store.LastConsensusEventFrom(p)
			if err != nil {
				return Frame{}, err
			}
			if isRoot {
				root, _ = h.Store.GetRoot(p)
			} else {
				lastConsensusEvent, err := h.Store.GetEvent(lastConsensusEventHash)
				if err != nil {
					return Frame{}, err
				}
				root, err = h.createRoot(lastConsensusEvent)
				if err != nil {
					return Frame{}, err
				}
			}
			roots[p] = root
		}
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
				if ev.SelfParent() != roots[ev.Creator()].SelfParent.Hash {
					other, err := h.createOtherParentRootEvent(ev)
					if err != nil {
						return Frame{}, err
					}
					roots[ev.Creator()].Others[ev.Hex()] = other
				}
			}
		}
	}

	//order roots
	orderedRoots := make([]Root, len(h.Participants))
	for i := 0; i < len(h.Participants); i++ {
		orderedRoots[i] = roots[h.ReverseParticipants[i]]
	}

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
				block.Index() > *h.AnchorBlock) {
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

	h.UndeterminedEvents = []string{}
	h.PendingRounds = []*pendingRound{}
	h.PendingLoadedEvents = 0
	h.topologicalIndex = 0

	cacheSize := h.Store.CacheSize()
	h.ancestorCache = common.NewLRU(cacheSize, nil)
	h.selfAncestorCache = common.NewLRU(cacheSize, nil)
	h.stronglySeeCache = common.NewLRU(cacheSize, nil)
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

//ReadWireInfo converts a WireEvent to an Event by replacing int IDs with the
//corresponding public keys.
func (h *Hashgraph) ReadWireInfo(wevent WireEvent) (*Event, error) {
	selfParent := rootSelfParent(wevent.Body.CreatorID)
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
			//PROBLEM Check if other parent can be found in the root
			//problem, we do not known the WireEvent's EventHash, and
			//we do not know the creators of the roots RootEvents
			root, err := h.Store.GetRoot(creator)
			if err != nil {
				return nil, err
			}
			//loop through others
			found := false
			for _, re := range root.Others {
				if re.CreatorID == wevent.Body.OtherParentCreatorID &&
					re.Index == wevent.Body.OtherParentIndex {
					otherParent = re.Hash
					found = true
					break
				}
			}

			if !found {
				return nil, fmt.Errorf("OtherParent not found")
			}

		}
	}

	body := EventBody{
		Transactions:    wevent.Body.Transactions,
		BlockSignatures: wevent.BlockSignatures(creatorBytes),
		Parents:         []string{selfParent, otherParent},
		Creator:         creatorBytes,

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

//CheckBlock returns an error if the Block does not contain valid signatures
//from MORE than 1/3 of participants
func (h *Hashgraph) CheckBlock(block Block) error {
	validSignatures := 0
	for _, s := range block.GetSignatures() {
		ok, _ := block.Verify(s)
		if ok {
			validSignatures++
		}
	}
	if validSignatures <= h.trustCount {
		return fmt.Errorf("Not enough valid signatures: got %d, need %d", validSignatures, h.trustCount+1)
	}

	h.logger.WithField("valid_signatures", validSignatures).Debug("CheckBlock")
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
