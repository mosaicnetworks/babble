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

type key struct {
	x, y string
}

func (k key) ToString() string {
	return fmt.Sprintf("{%s, %s}", k.x, k.y)
}

type treKey struct {
	x, y, z string
}

func (k treKey) ToString() string {
	return fmt.Sprintf("{%s, %s, %s}", k.x, k.y, k.z)
}

// ParticipantEventsCache is a cache associated with a peer-set that keeps a
// a RollingIndex of events for every peer.
type ParticipantEventsCache struct {
	participants *peers.PeerSet
	rim          *cm.RollingIndexMap
}

// NewParticipantEventsCache instantiates a new ParticipantEventsCache. The size
// parameter controls the size of each RollingIndex within the cache.
func NewParticipantEventsCache(size int) *ParticipantEventsCache {
	return &ParticipantEventsCache{
		participants: peers.NewPeerSet([]*peers.Peer{}),
		rim:          cm.NewRollingIndexMap("ParticipantEvents", size),
	}
}

// AddPeer adds a peer to the cache.
func (pec *ParticipantEventsCache) AddPeer(peer *peers.Peer) error {
	pec.participants = pec.participants.WithNewPeer(peer)
	return pec.rim.AddKey(peer.ID())
}

// particant is the CASE-INSENSITIVE string hex representation of the public
// key.
func (pec *ParticipantEventsCache) participantID(participant string) (uint32, error) {
	pUpper := strings.ToUpper(participant)
	peer, ok := pec.participants.ByPubKey[pUpper]
	if !ok {
		return 0, cm.NewStoreErr("ParticipantEvents", cm.UnknownParticipant, pUpper)
	}

	return peer.ID(), nil
}

// Get returns a participant's events with index > skip
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

// GetItem returns a specific event for a specific peer.
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

// GetLast returns the index of a participant's last event.
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

// Set attemps to set an event to a participant's events.
func (pec *ParticipantEventsCache) Set(participant string, hash string, index int) error {
	id, err := pec.participantID(participant)
	if err != nil {
		return err
	}
	return pec.rim.Set(id, hash, index)
}

// Known returns [participant id] => lastKnownIndex
func (pec *ParticipantEventsCache) Known() map[uint32]int {
	return pec.rim.Known()
}

// PeerSetCache is a cache that keeps track of peer-sets.
type PeerSetCache struct {
	rounds             sort.IntSlice
	peerSets           map[int]*peers.PeerSet
	repertoireByPubKey map[string]*peers.Peer
	repertoireByID     map[uint32]*peers.Peer
	firstRounds        map[uint32]int
}

// NewPeerSetCache creates a new PeerSetCache.
func NewPeerSetCache() *PeerSetCache {
	return &PeerSetCache{
		rounds:             sort.IntSlice{},
		peerSets:           make(map[int]*peers.PeerSet),
		repertoireByPubKey: make(map[string]*peers.Peer),
		repertoireByID:     make(map[uint32]*peers.Peer),
		firstRounds:        make(map[uint32]int),
	}
}

// Set adds a peer-set at a given round and updates internal information.
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

// Get returns the peer-set corresponding to a given round.
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

// GetAll returns all peer-sets in a map of round to peer-set.
func (c *PeerSetCache) GetAll() (map[int][]*peers.Peer, error) {
	res := make(map[int][]*peers.Peer)
	for _, r := range c.rounds {
		res[r] = c.peerSets[r].Peers
	}
	return res, nil
}

// RepertoireByID returns all the known peers indexed by ID. This includes peers
// that are no longer in the active peer-set.
func (c *PeerSetCache) RepertoireByID() map[uint32]*peers.Peer {
	return c.repertoireByID
}

// RepertoireByPubKey returns all the known peers indexed by public key.
func (c *PeerSetCache) RepertoireByPubKey() map[string]*peers.Peer {
	return c.repertoireByPubKey
}

// FirstRound returns the index of the first round where a peer appeared.
func (c *PeerSetCache) FirstRound(id uint32) (int, bool) {
	fr, ok := c.firstRounds[id]
	if ok {
		return fr, true
	}
	return math.MaxInt32, false
}

// PendingRound represents a round as it goes through consensus.
type PendingRound struct {
	Index   int
	Decided bool
}

// OrderedPendingRounds is an ordered list of PendingRounds.
type OrderedPendingRounds []*PendingRound

// Len returns the length
func (a OrderedPendingRounds) Len() int { return len(a) }

// Swap swaps 2 elements
func (a OrderedPendingRounds) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less returns true if element i is less than element j.
func (a OrderedPendingRounds) Less(i, j int) bool {
	return a[i].Index < a[j].Index
}

// PendingRoundsCache is a cache for PendingRounds.
type PendingRoundsCache struct {
	items       map[int]*PendingRound
	sortedItems OrderedPendingRounds
}

// NewPendingRoundsCache creates a new PendingRoundsCache.
func NewPendingRoundsCache() *PendingRoundsCache {
	return &PendingRoundsCache{
		items:       make(map[int]*PendingRound),
		sortedItems: []*PendingRound{},
	}
}

// Queued indicates whether a round is already part of the PendingRoundsCache.
func (c *PendingRoundsCache) Queued(round int) bool {
	_, ok := c.items[round]
	return ok
}

// Set adds an item to the PendingRoundCache and preserves the order.
func (c *PendingRoundsCache) Set(pendingRound *PendingRound) {
	c.items[pendingRound.Index] = pendingRound
	c.sortedItems = append(c.sortedItems, pendingRound)
	sort.Sort(c.sortedItems)
}

// GetOrderedPendingRounds returns the ordered list of PendingRounds.
func (c *PendingRoundsCache) GetOrderedPendingRounds() OrderedPendingRounds {
	return c.sortedItems
}

// Update takes a list of indexes of rounds that have been decided and updates
// the PendingRoundsCache accordingly.
func (c *PendingRoundsCache) Update(decidedRounds []int) {
	for _, drn := range decidedRounds {
		if dr, ok := c.items[drn]; ok {
			dr.Decided = true
		}
	}
}

// Clean removes a list of rounds from the cache and preserves the sorted order.
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

// SigPool holds a collection of BlockSignatures.
type SigPool struct {
	items map[string]BlockSignature
}

// NewSigPool creates a new SigPool.
func NewSigPool() *SigPool {
	return &SigPool{
		items: make(map[string]BlockSignature),
	}
}

// Add adds an item to SigPool.
func (sp *SigPool) Add(blockSignature BlockSignature) {
	sp.items[blockSignature.Key()] = blockSignature
}

// Remove removes an item from SigPool.
func (sp *SigPool) Remove(key string) {
	delete(sp.items, key)
}

// RemoveSlice removes multiple items from SigPool.
func (sp *SigPool) RemoveSlice(sigs []BlockSignature) {
	for _, s := range sigs {
		delete(sp.items, s.Key())
	}
}

// Len returns the number of items in the SigPool.
func (sp *SigPool) Len() int {
	return len(sp.items)
}

// Items returns the contents of the SigPool as a map.
func (sp *SigPool) Items() map[string]BlockSignature {
	return sp.items
}

// Slice returns the contents of the SigPool as a slice in no particular order.
func (sp *SigPool) Slice() []BlockSignature {
	res := []BlockSignature{}
	for _, bs := range sp.items {
		res = append(res, bs)
	}
	return res
}
