package hashgraph

import (
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

const (
	/*
		ROOT_DEPTH determines how many RootEvents are included in the Root.Past
		map. It is preferable not to make ROOT_DEPTH configurable because if
		peers use diffent values, they will produce different Roots, different
		Frames, and different Blocks. Perhaps this parameter should be tied to
		the number of Peers rather than hard-coded.
	*/
	ROOT_DEPTH = 10

	/*
		COIN_ROUND_FREQ defines the frequency of coin rounds. The value 4 is
		arbitrary. Do something smarter.
	*/
	COIN_ROUND_FREQ = float64(4)
)

//Hashgraph is a DAG of Events. It also contains methods to extract a consensus
//order of Events and map them onto a blockchain.
type Hashgraph struct {
	Store                   Store                  //store of Events, Rounds, and Blocks
	UndeterminedEvents      []string               //[index] => hash . FIFO queue of Events whose consensus order is not yet determined
	PendingRounds           *PendingRoundsCache    //FIFO queue of Rounds which have not attained consensus yet
	PendingSignatures       *SigPool               //Pool of Block signatures that need to be processed (matched with Blocks)
	LastConsensusRound      *int                   //index of last consensus round
	FirstConsensusRound     *int                   //index of first consensus round (only used in tests)
	AnchorBlock             *int                   //index of last block with enough signatures
	roundLowerBound         *int                   //rounds and events below this lower bound have a special treatement (cf fastsync)
	LastCommitedRoundEvents int                    //number of events in round before LastConsensusRound
	ConsensusTransactions   int                    //number of consensus transactions
	PendingLoadedEvents     int                    //number of loaded events that are not yet committed
	commitCallback          InternalCommitCallback //commit block callback
	topologicalIndex        int                    //counter used to order events in topological order (only local)

	ancestorCache     *common.LRU
	selfAncestorCache *common.LRU
	stronglySeeCache  *common.LRU
	roundCache        *common.LRU
	timestampCache    *common.LRU

	logger *logrus.Entry
}

//NewHashgraph instantiates a Hashgraph with an underlying data store and a
//commit callback
func NewHashgraph(store Store, commitCallback InternalCommitCallback, logger *logrus.Entry) *Hashgraph {
	if logger == nil {
		log := logrus.New()
		log.Level = logrus.DebugLevel
		logger = logrus.NewEntry(log)
	}

	cacheSize := store.CacheSize()
	hashgraph := Hashgraph{
		Store:             store,
		PendingRounds:     NewPendingRoundsCache(),
		PendingSignatures: NewSigPool(),
		commitCallback:    commitCallback,
		ancestorCache:     common.NewLRU(cacheSize, nil),
		selfAncestorCache: common.NewLRU(cacheSize, nil),
		stronglySeeCache:  common.NewLRU(cacheSize, nil),
		roundCache:        common.NewLRU(cacheSize, nil),
		timestampCache:    common.NewLRU(cacheSize, nil),
		logger:            logger,
	}

	return &hashgraph
}

//Init sets the initial PeerSet, which also creates the corresponding Roots and
//updates the Repertoire.
func (h *Hashgraph) Init(peerSet *peers.PeerSet) error {
	if err := h.Store.SetPeerSet(0, peerSet); err != nil {
		return fmt.Errorf("Error setting PeerSet: %v", err)
	}

	//XXX Do something else? Genesis Block?

	return nil
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

	entry, ok := ex.lastAncestors[ey.Creator()]

	res := ok && entry.index >= ey.Index()

	return res, nil
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

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false, err
	}

	return ex.Creator() == ey.Creator() && ex.Index() >= ey.Index(), nil
}

/*
True if x sees y

It is not necessary to detect forks because we assume that the InsertEvent
function makes it impossible to insert two Events at the same height for
the same participant.
*/
func (h *Hashgraph) see(x, y string) (bool, error) {
	return h.ancestor(x, y)
}

//true if x strongly sees y based on peers set
func (h *Hashgraph) stronglySee(x, y string, peers *peers.PeerSet) (bool, error) {
	if c, ok := h.stronglySeeCache.Get(TreKey{x, y, peers.Hex()}); ok {
		return c.(bool), nil
	}
	ss, err := h._stronglySee(x, y, peers)
	if err != nil {
		return false, err
	}
	h.stronglySeeCache.Add(TreKey{x, y, peers.Hex()}, ss)
	return ss, nil
}

func (h *Hashgraph) _stronglySee(x, y string, peers *peers.PeerSet) (bool, error) {

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return false, err
	}

	ey, err := h.Store.GetEvent(y)
	if err != nil {
		return false, err
	}

	c := 0
	for p := range peers.ByPubKey {
		xla, xlaok := ex.lastAncestors[p]
		yfd, yfdok := ey.firstDescendants[p]
		if xlaok && yfdok && xla.index >= yfd.index {
			c++
		}
	}

	return c >= peers.SuperMajority(), nil
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
	//x is the Root's Head; use Root.Head.Round
	rootsBySelfParent := h.Store.RootsBySelfParent()
	if r, ok := rootsBySelfParent[x]; ok {
		return r.GetHead().Round, nil
	}

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return math.MinInt32, err
	}

	root, err := h.Store.GetRoot(ex.Creator())
	if err != nil {
		return math.MinInt32, err
	}

	//x is attached to the Root; use pre-computed value.
	if v, ok := root.Precomputed[x]; ok {
		return v.Round, nil
	}

	//The Event's parents are "normal" Events; use the whitepaper formula:
	//parentRound + roundInc
	parentRound, err := h.round(ex.SelfParent())
	if err != nil {
		return math.MinInt32, err
	}

	if ex.OtherParent() != "" {
		opRound, err := h.round(ex.OtherParent())
		if err != nil {
			return math.MinInt32, err
		}

		if opRound > parentRound {
			parentRound = opRound
		}
	}

	/*
		Retrieve the ParentRound's PeerSet and count strongly-seen witnesses
		based on this PeerSet. If the ParentRound, or its PeerSet are not found,
		we just assume that the Round should be incremented (this situation
		occurs with a Reset hashgraph).
	*/
	parentRoundObj, err := h.Store.GetRound(parentRound)
	if err != nil {
		if common.Is(err, common.KeyNotFound) {
			return parentRound + 1, nil
		}
		return math.MinInt32, err
	}

	parentRoundPeerSet, err := h.Store.GetPeerSet(parentRound)
	if err != nil {
		if common.Is(err, common.KeyNotFound) {
			return parentRound + 1, nil
		}
		return math.MinInt32, err
	}

	c := 0
	for _, w := range parentRoundObj.Witnesses() {
		ss, err := h.stronglySee(x, w, parentRoundPeerSet)
		if err != nil {
			return math.MinInt32, err
		}
		if ss {
			c++
		}
	}

	if c >= parentRoundPeerSet.SuperMajority() {
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

	//does the creator belong to the PeerSet?
	peerSet, err := h.Store.GetPeerSet(xRound)
	if err != nil {
		return false, err
	}

	if _, ok := peerSet.ByPubKey[ex.Creator()]; !ok {
		return false, nil
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
	//x is the Root's Head; use Root.Head.LamportTimestamp
	rootsBySelfParent := h.Store.RootsBySelfParent()
	if r, ok := rootsBySelfParent[x]; ok {
		return r.GetHead().LamportTimestamp, nil
	}

	ex, err := h.Store.GetEvent(x)
	if err != nil {
		return math.MinInt32, err
	}

	root, err := h.Store.GetRoot(ex.Creator())
	if err != nil {
		return math.MinInt32, err
	}

	if v, ok := root.Precomputed[x]; ok {
		return v.LamportTimestamp, nil
	}

	plt, err := h.lamportTimestamp(ex.SelfParent())
	if err != nil {
		return math.MinInt32, err
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
func (h *Hashgraph) checkSelfParent(event *Event) error {
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
func (h *Hashgraph) checkOtherParent(event *Event) error {
	otherParent := event.OtherParent()
	if otherParent != "" {
		//Check if we have it
		_, err := h.Store.GetEvent(otherParent)
		if err != nil {
			//could be in the Root's PastEvents
			for p := range h.Store.RepertoireByPubKey() {
				root, err := h.Store.GetRoot(p)
				if err != nil {
					return err
				}
				if _, ok := root.Past[otherParent]; ok {
					return nil
				}
			}

			return fmt.Errorf("Other-parent not known")
		}
	}
	return nil
}

//initialize arrays of last ancestors and first descendants
func (h *Hashgraph) initEventCoordinates(event *Event) error {
	event.lastAncestors = NewCoordinatesMap()
	event.firstDescendants = NewCoordinatesMap()

	selfParent, selfParentError := h.Store.GetEvent(event.SelfParent())
	otherParent, otherParentError := h.Store.GetEvent(event.OtherParent())

	if selfParentError != nil && otherParentError == nil {
		event.lastAncestors = otherParent.lastAncestors.Copy()
	} else if otherParentError != nil && selfParentError == nil {
		event.lastAncestors = selfParent.lastAncestors.Copy()
	} else if otherParentError == nil && selfParentError == nil {
		selfParentLastAncestors := selfParent.lastAncestors
		otherParentLastAncestors := otherParent.lastAncestors

		event.lastAncestors = selfParentLastAncestors.Copy()
		for p, ola := range otherParentLastAncestors {
			sla, ok := event.lastAncestors[p]
			if !ok || sla.index < ola.index {
				event.lastAncestors[p] = EventCoordinates{
					index: ola.index,
					hash:  ola.hash,
				}
			}
		}
	}

	event.firstDescendants[event.Creator()] = EventCoordinates{
		index: event.Index(),
		hash:  event.Hex(),
	}

	event.lastAncestors[event.Creator()] = EventCoordinates{
		index: event.Index(),
		hash:  event.Hex(),
	}

	return nil
}

//update first decendant of each last ancestor to point to event
func (h *Hashgraph) updateAncestorFirstDescendant(event *Event) error {
	for _, c := range event.lastAncestors {
		ah := c.hash
		for {
			a, err := h.Store.GetEvent(ah)
			if err != nil {
				break
			}

			_, ok := a.firstDescendants[event.Creator()]
			if !ok {
				a.firstDescendants[event.Creator()] = EventCoordinates{
					index: event.Index(),
					hash:  event.Hex(),
				}
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

func (h *Hashgraph) createRootEvent(x string) (RootEvent, error) {
	ev, err := h.Store.GetEvent(x)
	if err != nil {
		//it might still be in the Roots
		for p := range h.Store.RepertoireByPubKey() {
			r, err := h.Store.GetRoot(p)
			if err != nil {
				return RootEvent{}, err
			}
			if pe, ok := r.Past[x]; ok {
				return pe, nil
			}
		}
		return RootEvent{}, fmt.Errorf("RootEvent %s not found", x)
	}

	round, err := h.round(x)
	if err != nil {
		return RootEvent{}, err
	}

	lt, err := h.lamportTimestamp(x)
	if err != nil {
		return RootEvent{}, err
	}

	rootEvent := RootEvent{
		Index:            ev.Index(),
		CreatorID:        h.Store.RepertoireByPubKey()[ev.Creator()].ID(),
		Hash:             ev.Hex(),
		Round:            round,
		LamportTimestamp: lt,
	}

	return rootEvent, nil
}

func (h *Hashgraph) createRoot(participant string, head string) (*Root, error) {
	headEvent, err := h.createRootEvent(head)
	if err != nil {
		return nil, err
	}

	root := NewRoot(headEvent)

	oldRoot, err := h.Store.GetRoot(participant)
	if err != nil {
		return nil, err
	}

	index := headEvent.Index
	for i := 0; i < ROOT_DEPTH; i++ {
		index = index - 1
		if index >= 0 {
			if pe, ok := oldRoot.PastByIndex(index); ok {
				root.Insert(pe)
			} else {
				peh, err := h.Store.ParticipantEvent(participant, index)
				if err != nil {
					break
				}
				rev, err := h.createRootEvent(peh)
				if err != nil {
					return nil, err
				}
				root.Insert(rev)
			}
		} else {
			break
		}
	}

	return root, nil
}

func (h *Hashgraph) setWireInfo(event *Event) error {
	selfParentIndex := -1
	otherParentCreatorID := uint32(0)
	otherParentIndex := -1

	creator, ok := h.Store.RepertoireByPubKey()[event.Creator()]
	if !ok {
		return fmt.Errorf("Creator %s not found", event.Creator())
	}

	//could be the first Event inserted for this creator. In this case, use Root
	if lf, isRoot, _ := h.Store.LastEventFrom(creator.PubKeyHex); isRoot && lf == event.SelfParent() {
		root, err := h.Store.GetRoot(creator.PubKeyHex)
		if err != nil {
			return err
		}
		selfParentIndex = root.GetHead().Index
	} else {
		selfParent, err := h.Store.GetEvent(event.SelfParent())
		if err != nil {
			return err
		}
		selfParentIndex = selfParent.Index()
	}

	if event.OtherParent() != "" {
		//Check Root's PastEvents, then regular Events
		foundInRoots := false
		for p := range h.Store.RepertoireByPubKey() {
			root, err := h.Store.GetRoot(p)
			if err != nil {
				return err
			}
			if pe, ok := root.Past[event.OtherParent()]; ok {
				otherParentCreatorID = pe.CreatorID
				otherParentIndex = pe.Index
				foundInRoots = true
				break
			}
		}

		if !foundInRoots {
			otherParent, err := h.Store.GetEvent(event.OtherParent())
			if err != nil {
				return err
			}
			otherParentCreator, ok := h.Store.RepertoireByPubKey()[otherParent.Creator()]
			if !ok {
				return fmt.Errorf("Creator %s not found", otherParent.Creator())
			}
			otherParentCreatorID = otherParentCreator.ID()
			otherParentIndex = otherParent.Index()
		}
	}

	event.SetWireInfo(selfParentIndex,
		otherParentCreatorID,
		otherParentIndex,
		creator.ID())

	return nil
}

//Remove processed Signatures from SigPool
func (h *Hashgraph) removeProcessedSignatures(processedSignatures map[string]bool) {
	for k := range processedSignatures {
		h.PendingSignatures.Remove(k)
	}
}

/*******************************************************************************
Public Methods
*******************************************************************************/

//InsertEventAndRunConsensus inserts an Event in the Hashgraph and call the
//consensus methods.
func (h *Hashgraph) InsertEventAndRunConsensus(event *Event, setWireInfo bool) error {
	if err := h.InsertEvent(event, setWireInfo); err != nil {
		return err
	}
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
	return nil
}

//InsertEvent attempts to insert an Event in the DAG. It verifies the signature,
//checks the ancestors are known, and prevents the introduction of forks.
func (h *Hashgraph) InsertEvent(event *Event, setWireInfo bool) error {
	//XXX TODO
	//Check if creator belongs to round

	//verify signature
	if ok, err := event.Verify(); !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("Invalid Event signature")
	}

	if err := h.checkSelfParent(event); err != nil {
		h.logger.WithFields(logrus.Fields{
			"event":       event.Hex(),
			"creator":     event.Creator(),
			"self_parent": event.SelfParent(),
		}).WithError(err).Errorf("CheckSelfParent")
		return err
	}

	if err := h.checkOtherParent(event); err != nil {
		h.logger.WithFields(logrus.Fields{
			"event":        event.Hex(),
			"creator":      event.Creator(),
			"other_parent": event.OtherParent(),
		}).WithError(err).Errorf("CheckOtherParent")
		return err
	}

	event.topologicalIndex = h.topologicalIndex
	h.topologicalIndex++

	if setWireInfo {
		if err := h.setWireInfo(event); err != nil {
			return fmt.Errorf("SetWireInfo: %s", err)
		}
	}

	if err := h.initEventCoordinates(event); err != nil {
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

	for _, bs := range event.BlockSignatures() {
		h.logger.Debugf("Inserting pending signature %v", bs.Key())
		h.PendingSignatures.Add(bs)
	}

	return nil
}

//DivideRounds assigns a Round and LamportTimestamp to Events, and flags them as
//witnesses if necessary. Pushes Rounds in the PendingRounds queue if necessary.
func (h *Hashgraph) DivideRounds() error {
	for _, hash := range h.UndeterminedEvents {
		ev, err := h.Store.GetEvent(hash)
		if err != nil {
			return err
		}

		updateEvent := false

		//Compute Event's round, update the corresponding Round object, and
		//add it to the PendingRounds queue if necessary.
		if ev.round == nil {
			roundNumber, err := h.round(hash)
			if err != nil {
				return err
			}

			ev.SetRound(roundNumber)
			updateEvent = true

			roundInfo, err := h.Store.GetRound(roundNumber)
			if err != nil {
				if !common.Is(err, common.KeyNotFound) {
					return err
				}

				roundInfo = NewRoundInfo()
			}

			if !h.PendingRounds.Queued(roundNumber) && !roundInfo.decided {
				h.PendingRounds.Set(&PendingRound{roundNumber, false})
			}

			witness, err := h.witness(hash)
			if err != nil {
				return err
			}

			roundInfo.AddCreatedEvent(hash, witness)

			err = h.Store.SetRound(roundNumber, roundInfo)
			if err != nil {
				return err
			}
		}

		//Compute the Event's LamportTimestamp
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

	decidedRounds := []int{}

	for _, r := range h.PendingRounds.GetOrderedPendingRounds() {
		roundIndex := r.Index

		rRoundInfo, err := h.Store.GetRound(roundIndex)
		if err != nil {
			return err
		}

		rPeerSet, err := h.Store.GetPeerSet(roundIndex)
		if err != nil {
			return err
		}

		for _, x := range rRoundInfo.Witnesses() {
			if rRoundInfo.IsDecided(x) {
				continue
			}
		VOTE_LOOP:
			for j := roundIndex + 1; j <= h.Store.LastRound(); j++ {
				jRoundInfo, err := h.Store.GetRound(j)
				if err != nil {
					//This can happen in FastSync
					break
				}

				jPeerSet, err := h.Store.GetPeerSet(j)
				if err != nil {
					return err
				}

				for _, y := range jRoundInfo.Witnesses() {
					diff := j - roundIndex
					if diff == 1 {
						ycx, err := h.see(y, x)
						if err != nil {
							return err
						}
						setVote(votes, y, x, ycx)
					} else {
						jPrevRoundInfo, err := h.Store.GetRound(j - 1)
						if err != nil {
							return err
						}

						jPrevPeerSet, err := h.Store.GetPeerSet(j - 1)
						if err != nil {
							return err
						}

						//collection of witnesses from round j-1 that are
						//strongly seen by y, based on round j-1 PeerSet.
						ssWitnesses := []string{}
						for _, w := range jPrevRoundInfo.Witnesses() {
							ss, err := h.stronglySee(y, w, jPrevPeerSet)
							if err != nil {
								return err
							}
							if ss {
								ssWitnesses = append(ssWitnesses, w)
							}
						}

						//Collect votes from these witnesses.
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

						//float64(rPeerSet.Len())
						//normal round
						if math.Mod(float64(diff), COIN_ROUND_FREQ) > 0 {
							if t >= jPeerSet.SuperMajority() {
								rRoundInfo.SetFame(x, v)
								setVote(votes, y, x, v)
								break VOTE_LOOP //break out of j loop
							} else {
								setVote(votes, y, x, v)
							}
						} else { //coin round
							if t >= jPeerSet.SuperMajority() {
								setVote(votes, y, x, v)
							} else {
								setVote(votes, y, x, middleBit(y)) //middle bit of y's hash
							}
						}
					}
				}
			}
		}

		if rRoundInfo.WitnessesDecided(rPeerSet) {
			decidedRounds = append(decidedRounds, roundIndex)
		}

		err = h.Store.SetRound(roundIndex, rRoundInfo)
		if err != nil {
			return err
		}
	}

	h.PendingRounds.Update(decidedRounds)
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
				//This can happen in FastSync
				break
			}

			tPeers, err := h.Store.GetPeerSet(i)
			if err != nil {
				return err
			}

			/*
				We are looping from earlier to later rounds; so if we encounter
				one round with undecided witnesses, we are sure that this event
				is not "received". Break out of i loop. This is not true when
				when Resetting from a Frame; it is possiblie that we just don't
				have all the round's witnesses. In this case, just continue
				through the i loop.
			*/
			if !(tr.WitnessesDecided(tPeers)) {
				if h.roundLowerBound == nil || *h.roundLowerBound < i {
					break
				} else {
					continue
				}
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

			if len(s) == len(fws) && len(s) >= tPeers.SuperMajority() {
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

				tr.AddReceivedEvent(x)
				err = h.Store.SetRound(i, tr)
				if err != nil {
					return err
				}

				//Only count loaded Events of rounds above the lower bound
				if h.roundLowerBound != nil && *h.roundLowerBound >= i && ex.IsLoaded() {
					h.PendingLoadedEvents--
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

/*
ProcessDecidedRounds takes Rounds whose witnesses are decided, computes the
corresponding Frames, maps them into Blocks, and commits the Blocks via the
commit channel
*/
func (h *Hashgraph) ProcessDecidedRounds() error {
	//Defer removing processed Rounds from the PendingRounds Queue
	processedRounds := []int{}
	defer func() {
		h.PendingRounds.Clean(processedRounds)
	}()

	for _, r := range h.PendingRounds.GetOrderedPendingRounds() {
		/*
			After a Reset, round roundLowerBound (=LastConsensusRound) is added
			to PendingRounds, but its ConsensusEvents (which are necessarily
			'under' this Round) are already deemed committed. Hence, skip this
			Round after a Reset.
		*/
		if h.roundLowerBound != nil && r.Index <= *h.roundLowerBound {
			h.logger.WithField("round_received", r.Index).Debug("Skipping Pending Round")
			continue
		}

		/*
			Although it is possible for a Round to be 'decided' before a previous
			round, we should NEVER process a decided round before all the earlier
			rounds are processed.
		*/
		if !r.Decided {
			break
		}

		round, err := h.Store.GetRound(r.Index)
		if err != nil {
			return err
		}

		frame, err := h.GetFrame(r.Index)
		if err != nil {
			return fmt.Errorf("Getting Frame %d: %v", r.Index, err)
		}

		h.logger.WithFields(logrus.Fields{
			"round_received":  r.Index,
			"witnesses":       round.FamousWitnesses(),
			"created_events":  round.CreatedEvents,
			"events":          len(frame.Events),
			"peers":           len(frame.Peers),
			"future_peersets": frame.FuturePeerSets,
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

			if len(block.Transactions()) > 0 ||
				len(block.InternalTransactions()) > 0 {

				if err := h.Store.SetBlock(block); err != nil {
					return err
				}

				err := h.commitCallback(block)
				if err != nil {
					h.logger.Warningf("Failed to commit block %d", block.Index())
				}
			}
		} else {
			h.logger.Debugf("No Events to commit for ConsensusRound %d", r.Index)
		}

		processedRounds = append(processedRounds, r.Index)

		if h.LastConsensusRound == nil || r.Index > *h.LastConsensusRound {
			h.setLastConsensusRound(r.Index)
		}
	}

	return nil
}

//GetFrame computes the Frame corresponding to a RoundReceived.
func (h *Hashgraph) GetFrame(roundReceived int) (*Frame, error) {
	//Try to get it from the Store first
	frame, err := h.Store.GetFrame(roundReceived)
	if err == nil || !common.Is(err, common.KeyNotFound) {
		return frame, err
	}

	//Get the Round and corresponding consensus Events
	round, err := h.Store.GetRound(roundReceived)
	if err != nil {
		return nil, err
	}

	peerSet, err := h.Store.GetPeerSet(roundReceived)
	if err != nil {
		return nil, err
	}

	//Get Future PeerSets
	futurePeerSets, err := h.Store.GetFuturePeerSets(roundReceived)
	if err != nil {
		return nil, err
	}

	events := []*Event{}
	for _, eh := range round.ReceivedEvents {
		e, err := h.Store.GetEvent(eh)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	sort.Sort(ByLamportTimestamp(events))

	/*
		Get/Create Roots. The events are in topological order; so each time we
		run into the first Event of a participant, we create a Root for it. Then
		we populate the root's Precomputed map.
	*/
	roots := make(map[string]*Root)

	for _, ev := range events {
		p := ev.Creator()
		r, ok := roots[p]
		if !ok {
			r, err = h.createRoot(p, ev.SelfParent())
			if err != nil {
				return nil, err
			}
			roots[p] = r
		}
		rootEvent, err := h.createRootEvent(ev.Hex())
		if err != nil {
			return nil, err
		}
		r.Precomputed[ev.Hex()] = rootEvent
	}
	/*
		Every participant needs a Root in the Frame. For the participants that
		have no Events in this Frame, we create a Root from their last consensus
		Event, or their last known Root
	*/
	for _, p := range peerSet.PubKeys() {
		if _, ok := roots[p]; !ok {
			var root *Root
			lastConsensusEventHash, isRoot, err := h.Store.LastConsensusEventFrom(p)
			if err != nil {
				return nil, err
			}
			if isRoot {
				root, _ = h.Store.GetRoot(p)
			} else {
				root, err = h.createRoot(p, lastConsensusEventHash)
				if err != nil {
					return nil, err
				}
			}
			roots[p] = root
		}
	}

	/*
		Some Events in the Frame might have other-parents that are outside of
		the Frame (cf root.go ex 2), and outside of the Root's PastEvents. When
		inserting these Events in a newly reset hashgraph, the CheckOtherParent
		method would return an error because the other-parent would not be
		found. So we make it possible to also look for other-parents in the
		Roots.
	*/
	treated := map[string]bool{}
	for _, ev := range events {
		treated[ev.Hex()] = true
		otherParent := ev.OtherParent()
		if otherParent != "" {
			opt, ok := treated[otherParent]
			if !opt || !ok {
				other, err := h.createRootEvent(ev.OtherParent())
				if err != nil {
					return nil, err
				}
				otherCreator := h.Store.RepertoireByID()[other.CreatorID]
				roots[otherCreator.PubKeyHex].Insert(other)

			}
		}
	}

	res := &Frame{
		Round:          roundReceived,
		Peers:          peerSet.Peers,
		Roots:          roots,
		Events:         events,
		FuturePeerSets: futurePeerSets,
	}

	if err := h.Store.SetFrame(res); err != nil {
		return nil, err
	}

	return res, nil
}

/*
ProcessSigPool runs through the SignaturePool and tries to map a Signature to
a known Block. If a Signature is valid, it is appended to the block and removed
from the SignaturePool. The function also updates the AnchorBlock if necessary.
*/
func (h *Hashgraph) ProcessSigPool() error {
	h.logger.WithField("pending_signatures", h.PendingSignatures.Len()).Debug("ProcessSigPool()")

	for _, bs := range h.PendingSignatures.Items() {
		block, err := h.Store.GetBlock(bs.Index)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"index": bs.Index,
				"msg":   err,
			}).Warning("Verifying Block signature. Could not fetch Block")
			continue
		}

		peerSet, err := h.Store.GetPeerSet(block.RoundReceived())
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"index": bs.Index,
				"round": block.RoundReceived(),
				"err":   err,
			}).Warning("Verifying Block signature. No PeerSet for Block's Round ")
			continue
		}

		//check if validator belongs to list of participants
		if _, ok := peerSet.ByPubKey[bs.ValidatorHex()]; !ok {
			h.logger.WithFields(logrus.Fields{
				"index":     bs.Index,
				"validator": bs.ValidatorHex(),
			}).Warning("Verifying Block signature. Unknown validator")
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
				"validator": peerSet.ByPubKey[bs.ValidatorHex()],
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

		if err := h.SetAnchorBlock(block); err != nil {
			return err
		}

		h.logger.Debugf("processed sig %v", bs.Key())

		h.PendingSignatures.Remove(bs.Key())
	}

	return nil
}

/*
SetAnchorBlock sets the AnchorBlock index if the proposed block has collected
enough signatures (+1/3) and is above the current AnchorBlock. The AnchorBlock
is the latest Block that collected +1/3 signatures from validators. It is used
in FastForward responses when a node wants to sync to the top of the hashgraph.
*/
func (h *Hashgraph) SetAnchorBlock(block *Block) error {
	peerSet, err := h.Store.GetPeerSet(block.RoundReceived())
	if err != nil {
		h.logger.WithError(err).Error("No PeerSet for Block's Round ")
		return err
	}

	if len(block.Signatures) > peerSet.TrustCount() &&
		(h.AnchorBlock == nil ||
			block.Index() > *h.AnchorBlock) {

		h.setAnchorBlock(block.Index())
		h.logger.WithFields(logrus.Fields{
			"block_index": block.Index(),
			"signatures":  len(block.Signatures),
			"trustCount":  peerSet.TrustCount(),
		}).Debug("Setting AnchorBlock")
	} else {
		h.logger.WithFields(logrus.Fields{
			"index":       block.Index(),
			"sigs":        len(block.Signatures),
			"trust_count": peerSet.TrustCount(),
		}).Debug("Block is not a suitable Anchor")
	}

	return nil
}

//GetAnchorBlockWithFrame returns the AnchorBlock and the corresponding Frame.
//This can be used as a base to Reset a Hashgraph
func (h *Hashgraph) GetAnchorBlockWithFrame() (*Block, *Frame, error) {
	if h.AnchorBlock == nil {
		return nil, nil, fmt.Errorf("No Anchor Block")
	}

	block, err := h.Store.GetBlock(*h.AnchorBlock)
	if err != nil {
		return nil, nil, err
	}

	frame, err := h.GetFrame(block.RoundReceived())
	if err != nil {
		return nil, nil, err
	}

	return block, frame, nil
}

//Reset clears the Hashgraph and resets it from a new base.
func (h *Hashgraph) Reset(block *Block, frame *Frame) error {
	//Clear all state
	h.LastConsensusRound = nil
	h.FirstConsensusRound = nil
	h.AnchorBlock = nil

	h.UndeterminedEvents = []string{}
	h.PendingRounds = NewPendingRoundsCache()
	h.PendingLoadedEvents = 0
	h.topologicalIndex = 0

	cacheSize := h.Store.CacheSize()
	h.ancestorCache = common.NewLRU(cacheSize, nil)
	h.selfAncestorCache = common.NewLRU(cacheSize, nil)
	h.stronglySeeCache = common.NewLRU(cacheSize, nil)
	h.roundCache = common.NewLRU(cacheSize, nil)

	//Initialize new Roots
	if err := h.Store.Reset(frame); err != nil {
		return err
	}

	//Insert Block
	if err := h.Store.SetBlock(block); err != nil {
		return err
	}

	h.setLastConsensusRound(block.RoundReceived())

	h.setRoundLowerBound(block.RoundReceived())

	//Insert Frame Events
	for _, ev := range frame.Events {
		if err := h.InsertEventAndRunConsensus(ev, false); err != nil {
			return err
		}
	}

	return nil
}

/*
Bootstrap loads all Events from the Store's DB (if there is one) and feeds
them to the Hashgraph (in topological order) for consensus ordering. After this
method call, the Hashgraph should be in a state coherent with the 'tip' of the
Hashgraph
*/
func (h *Hashgraph) Bootstrap() error {
	if badgerStore, ok := h.Store.(*BadgerStore); ok {
		//Load Genesis PeerSet
		peerSet, err := badgerStore.dbGetPeerSet(0)
		if err != nil {
			return fmt.Errorf("No Genesis PeerSet: %v", err)
		}

		//Initialize the InmemStore with Genesis PeerSet. This has side-effects:
		//It will create the corresponding Roots and populate the Repertoires.
		badgerStore.inmemStore.SetPeerSet(0, peerSet)

		//Retreive the Events from the underlying DB. They come out in topological
		//order
		topologicalEvents, err := badgerStore.dbTopologicalEvents()
		if err != nil {
			return err
		}

		//Insert the Events in the Hashgraph
		for _, e := range topologicalEvents {
			if err := h.InsertEventAndRunConsensus(e, true); err != nil {
				return err
			}
		}

		//ProcessSigPool
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

	creator, ok := h.Store.RepertoireByID()[wevent.Body.CreatorID]
	if !ok {
		return nil, fmt.Errorf("Creator %d not found", wevent.Body.CreatorID)
	}

	creatorBytes, err := hex.DecodeString(creator.PubKeyHex[2:])
	if err != nil {
		return nil, err
	}

	if wevent.Body.SelfParentIndex >= 0 {
		selfParent, err = h.Store.ParticipantEvent(creator.PubKeyHex, wevent.Body.SelfParentIndex)
		if err != nil {
			return nil, err
		}
	}

	if wevent.Body.OtherParentIndex >= 0 {
		otherParentCreator, ok := h.Store.RepertoireByID()[wevent.Body.OtherParentCreatorID]
		if !ok {
			return nil, fmt.Errorf("Participant %d not found", wevent.Body.OtherParentCreatorID)
		}

		otherParent, err = h.Store.ParticipantEvent(otherParentCreator.PubKeyHex, wevent.Body.OtherParentIndex)
		if err != nil {
			//Could be in OtherParent's Root PastEvents
			otherRoot, err := h.Store.GetRoot(otherParentCreator.PubKeyHex)
			if err != nil {
				return nil, err
			}

			pe, ok := otherRoot.PastByIndex(wevent.Body.OtherParentIndex)
			if !ok {
				return nil, fmt.Errorf("OtherParent (creator: %d, index: %d) not found", wevent.Body.OtherParentCreatorID, wevent.Body.OtherParentIndex)
			}

			otherParent = pe.Hash
		}
	}

	body := EventBody{
		Transactions:         wevent.Body.Transactions,
		InternalTransactions: wevent.Body.InternalTransactions,
		BlockSignatures:      wevent.BlockSignatures(creatorBytes),
		Parents:              []string{selfParent, otherParent},
		Creator:              creatorBytes,
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
func (h *Hashgraph) CheckBlock(block *Block, peerSet *peers.PeerSet) error {
	psh, err := peerSet.Hash()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(psh, block.PeersHash()) {
		return fmt.Errorf("Wrong PeerSet")
	}

	validSignatures := 0
	for _, s := range block.GetSignatures() {
		validatorHex := fmt.Sprintf("0x%X", s.Validator)
		if _, ok := peerSet.ByPubKey[validatorHex]; !ok {
			h.logger.WithFields(logrus.Fields{
				"validator": validatorHex,
			}).Warning("Verifying Block signature. Unknown validator")
			continue
		}
		ok, _ := block.Verify(s)
		if ok {
			validSignatures++
		}
	}

	if validSignatures <= peerSet.TrustCount() {
		return fmt.Errorf("Not enough valid signatures: got %d, need %d", validSignatures, peerSet.TrustCount())
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

func (h *Hashgraph) setRoundLowerBound(i int) {
	if h.roundLowerBound == nil {
		h.roundLowerBound = new(int)
	}
	*h.roundLowerBound = i
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

/*******************************************************************************
InternalCommitCallback
*******************************************************************************/

/*
InternalCommitCallback is called by the Hashgraph to commit a Block. The
InternalCommitCallback will likely itself call the ProxyCommitCallback. We add
a layer of indirection because processing the CommitResponse should be handled
by the Core object, not the hashgraph; the hashgraph only known if there was
an error or not.
*/
type InternalCommitCallback func(*Block) error

//DummyInternalCommitCallback is used for testing
func DummyInternalCommitCallback(b *Block) error {
	return nil
}
