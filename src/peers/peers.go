package peers

import (
	"sort"
	"sync"
)

type PubKeyPeers map[string]*Peer
type IdPeers map[int]*Peer

type Peers struct {
	sync.RWMutex
	Sorted   []*Peer
	ByPubKey PubKeyPeers
	ById     IdPeers
}

/* Constructors */

func NewPeers() *Peers {
	return &Peers{
		ByPubKey: make(PubKeyPeers),
		ById:     make(IdPeers),
	}
}

func NewPeersFromSlice(source []*Peer) *Peers {
	peers := NewPeers()

	for _, peer := range source {
		peers.addPeerRaw(peer)
	}

	peers.internalSort()

	return peers
}

/* Add Methods */

// Add a peer without sorting the set.
// Useful for adding a bunch of peers at the same time
// This method is private and is not protected by mutex.
// Handle with care
func (p *Peers) addPeerRaw(peer *Peer) {
	if peer.ID == 0 {
		peer.computeID()
	}

	p.ByPubKey[peer.PubKeyHex] = peer
	p.ById[peer.ID] = peer
}

func (p *Peers) AddPeer(peer *Peer) {
	p.Lock()
	defer p.Unlock()

	p.addPeerRaw(peer)

	p.internalSort()
}

func (p *Peers) internalSort() {
	res := []*Peer{}

	for _, p := range p.ByPubKey {
		res = append(res, p)
	}

	sort.Sort(ByID(res))

	p.Sorted = res
}

/* Remove Methods */

func (p *Peers) RemovePeer(peer *Peer) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.ByPubKey[peer.PubKeyHex]; !ok {
		return
	}

	delete(p.ByPubKey, peer.PubKeyHex)
	delete(p.ById, peer.ID)

	p.internalSort()
}

func (p *Peers) RemovePeerByPubKey(pubKey string) {
	p.RemovePeer(p.ByPubKey[pubKey])
}

func (p *Peers) RemovePeerById(id int) {
	p.RemovePeer(p.ById[id])
}

/* ToSlice Methods */

func (p *Peers) ToPeerSlice() []*Peer {
	return p.Sorted
}

func (p *Peers) ToPubKeySlice() []string {
	p.RLock()
	defer p.RUnlock()

	res := []string{}

	for _, peer := range p.Sorted {
		res = append(res, peer.PubKeyHex)
	}

	return res
}

func (p *Peers) ToIDSlice() []int {
	p.RLock()
	defer p.RUnlock()

	res := []int{}

	for _, peer := range p.Sorted {
		res = append(res, peer.ID)
	}

	return res
}

/* Utilities */

func (p *Peers) Len() int {
	p.RLock()
	defer p.RUnlock()

	return len(p.ByPubKey)
}

// ByPubHex implements sort.Interface for Peers based on
// the PubKeyHex field.
type ByPubHex []*Peer

func (a ByPubHex) Len() int      { return len(a) }
func (a ByPubHex) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPubHex) Less(i, j int) bool {
	ai := a[i].PubKeyHex
	aj := a[j].PubKeyHex
	return ai < aj
}

type ByID []*Peer

func (a ByID) Len() int      { return len(a) }
func (a ByID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByID) Less(i, j int) bool {
	ai := a[i].ID
	aj := a[j].ID
	return ai < aj
}
