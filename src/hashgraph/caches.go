package hashgraph

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	cm "github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
)

type Key struct {
	x, y string
}

func (k Key) ToString() string {
	return fmt.Sprintf("{%s, %s}", k.x, k.y)
}

type TreKey struct {
	x, y, z string
}

func (k TreKey) ToString() string {
	return fmt.Sprintf("{%s, %s, %s}", k.x, k.y, k.z)
}

//------------------------------------------------------------------------------

type ParticipantEventsCache struct {
	participants *peers.PeerSet
	rim          *cm.RollingIndexMap
}

func NewParticipantEventsCache(size int) *ParticipantEventsCache {
	return &ParticipantEventsCache{
		participants: peers.NewPeerSet([]*peers.Peer{}),
		rim:          cm.NewRollingIndexMap("ParticipantEvents", size),
	}
}

func (pec *ParticipantEventsCache) AddPeer(peer *peers.Peer) error {
	pec.participants = pec.participants.WithNewPeer(peer)
	return pec.rim.AddKey(peer.ID())
}

//particant is the CASE-INSENSITIVE string hex representation of the public key.
func (pec *ParticipantEventsCache) participantID(participant string) (uint32, error) {
	pUpper := strings.ToUpper(participant)
	peer, ok := pec.participants.ByPubKey[pUpper]
	if !ok {
		return 0, cm.NewStoreErr("ParticipantEvents", cm.UnknownParticipant, pUpper)
	}

	return peer.ID(), nil
}

//Get returns participant events with index > skip
func (pec *ParticipantEventsCache) Get(participant string, skipIndex int) ([]string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return []string{}, err
	}

	pe, err := pec.rim.Get(id, skipIndex)
	if err != nil {
		return []string{}, err
	}

	res := make([]string, len(pe))
	for k := 0; k < len(pe); k++ {
		res[k] = pe[k].(string)
	}
	return res, nil
}

func (pec *ParticipantEventsCache) GetItem(participant string, index int) (string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return "", err
	}

	item, err := pec.rim.GetItem(id, index)
	if err != nil {
		return "", err
	}
	return item.(string), nil
}

func (pec *ParticipantEventsCache) GetLast(participant string) (string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return "", err
	}

	last, err := pec.rim.GetLast(id)
	if err != nil {
		return "", err
	}

	return last.(string), nil
}

func (pec *ParticipantEventsCache) Set(participant string, hash string, index int) error {
	id, err := pec.participantID(participant)
	if err != nil {
		return err
	}
	return pec.rim.Set(id, hash, index)
}

//returns [participant id] => lastKnownIndex
func (pec *ParticipantEventsCache) Known() map[uint32]int {
	return pec.rim.Known()
}

//------------------------------------------------------------------------------

type PeerSetCache struct {
	rounds             sort.IntSlice
	peerSets           map[int]*peers.PeerSet
	repertoireByPubKey map[string]*peers.Peer
	repertoireByID     map[uint32]*peers.Peer
	firstRounds        map[uint32]int
}

func NewPeerSetCache() *PeerSetCache {
	return &PeerSetCache{
		rounds:             sort.IntSlice{},
		peerSets:           make(map[int]*peers.PeerSet),
		repertoireByPubKey: make(map[string]*peers.Peer),
		repertoireByID:     make(map[uint32]*peers.Peer),
		firstRounds:        make(map[uint32]int),
	}
}

func (c *PeerSetCache) Set(round int, peerSet *peers.PeerSet) error {
	if _, ok := c.peerSets[round]; ok {
		return cm.NewStoreErr("PeerSetCache", cm.KeyAlreadyExists, strconv.Itoa(round))
	}

	c.peerSets[round] = peerSet

	c.rounds = append(c.rounds, round)
	c.rounds.Sort()

	for _, p := range peerSet.Peers {
		c.repertoireByPubKey[p.PubKeyString()] = p
		c.repertoireByID[p.ID()] = p
		fr, ok := c.firstRounds[p.ID()]
		if !ok || fr > round {
			c.firstRounds[p.ID()] = round
		}
	}

	return nil

}

func (c *PeerSetCache) Get(round int) (*peers.PeerSet, error) {
	//check if directly in peerSets
	ps, ok := c.peerSets[round]
	if ok {
		return ps, nil
	}

	//situate round in sorted rounds
	if len(c.rounds) == 0 {
		return nil, cm.NewStoreErr("PeerSetCache", cm.KeyNotFound, strconv.Itoa(round))
	}

	if round < c.rounds[0] {
		return c.peerSets[c.rounds[0]], nil
	}

	for i := 0; i < len(c.rounds)-1; i++ {
		if round >= c.rounds[i] && round < c.rounds[i+1] {
			return c.peerSets[c.rounds[i]], nil
		}
	}

	//return last PeerSet
	return c.peerSets[c.rounds[len(c.rounds)-1]], nil
}

func (c *PeerSetCache) GetAll() (map[int][]*peers.Peer, error) {
	res := make(map[int][]*peers.Peer)
	for _, r := range c.rounds {
		res[r] = c.peerSets[r].Peers
	}
	return res, nil
}

func (c *PeerSetCache) RepertoireByID() map[uint32]*peers.Peer {
	return c.repertoireByID
}

func (c *PeerSetCache) RepertoireByPubKey() map[string]*peers.Peer {
	return c.repertoireByPubKey
}

func (c *PeerSetCache) FirstRound(id uint32) (int, bool) {
	fr, ok := c.firstRounds[id]
	if ok {
		return fr, true
	}
	return math.MaxInt32, false
}

//------------------------------------------------------------------------------

type PendingRound struct {
	Index   int
	Decided bool
}

type OrderedPendingRounds []*PendingRound

func (a OrderedPendingRounds) Len() int      { return len(a) }
func (a OrderedPendingRounds) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a OrderedPendingRounds) Less(i, j int) bool {
	return a[i].Index < a[j].Index
}

type PendingRoundsCache struct {
	items       map[int]*PendingRound
	sortedItems OrderedPendingRounds
}

func NewPendingRoundsCache() *PendingRoundsCache {
	return &PendingRoundsCache{
		items:       make(map[int]*PendingRound),
		sortedItems: []*PendingRound{},
	}
}

func (c *PendingRoundsCache) Queued(round int) bool {
	_, ok := c.items[round]
	return ok
}

func (c *PendingRoundsCache) Set(pendingRound *PendingRound) {
	c.items[pendingRound.Index] = pendingRound
	c.sortedItems = append(c.sortedItems, pendingRound)
	sort.Sort(c.sortedItems)
}

func (c *PendingRoundsCache) GetOrderedPendingRounds() OrderedPendingRounds {
	return c.sortedItems
}

func (c *PendingRoundsCache) Update(decidedRounds []int) {
	for _, drn := range decidedRounds {
		if dr, ok := c.items[drn]; ok {
			dr.Decided = true
		}
	}
}

func (c *PendingRoundsCache) Clean(processedRounds []int) {
	for _, pr := range processedRounds {
		delete(c.items, pr)
	}
	newSortedItems := OrderedPendingRounds{}
	for _, pr := range c.items {
		newSortedItems = append(newSortedItems, pr)
	}
	sort.Sort(newSortedItems)
	c.sortedItems = newSortedItems
}

//------------------------------------------------------------------------------

type SigPool struct {
	items map[string]BlockSignature
}

func NewSigPool() *SigPool {
	return &SigPool{
		items: make(map[string]BlockSignature),
	}
}

func (sp *SigPool) Add(blockSignature BlockSignature) {
	sp.items[blockSignature.Key()] = blockSignature
}

func (sp *SigPool) Remove(key string) {
	delete(sp.items, key)
}

func (sp *SigPool) RemoveSlice(sigs []BlockSignature) {
	for _, s := range sigs {
		delete(sp.items, s.Key())
	}
}

func (sp *SigPool) Len() int {
	return len(sp.items)
}

func (sp *SigPool) Items() map[string]BlockSignature {
	return sp.items
}

func (sp *SigPool) Slice() []BlockSignature {
	res := []BlockSignature{}
	for _, bs := range sp.items {
		res = append(res, bs)
	}
	return res
}
