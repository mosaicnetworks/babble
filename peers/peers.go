package peers

import "sort"

type PubKeyPeers map[string]*Peer
type IdPeers map[int]*Peer

type Peers struct {
	ByAppearance []int
	ByPubKey     PubKeyPeers
	ById         IdPeers
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
		peers.AddPeer(peer)
	}

	return peers
}

/* Add Methods */

func (p *Peers) AddPeer(peer *Peer) {
	// TODO: Check if peer doesn't exists
	if peer.ID == 0 {
		peer.computeID()
	}

	p.ByPubKey[peer.PubKeyHex] = peer
	p.ById[peer.ID] = peer
}

/* Remove Methods */

func (p *Peers) RemovePeer(peer *Peer) {
	// TODO: Check if peer exists
	delete(p.ByPubKey, peer.PubKeyHex)
	delete(p.ById, peer.ID)
}

func (p *Peers) RemovePeerByPubKey(pubKey string) {
	// TODO: Check if peer exists
	id := p.ByPubKey[pubKey].ID

	delete(p.ByPubKey, pubKey)
	delete(p.ById, id)
}

func (p *Peers) RemovePeerById(id int) {
	// TODO: Check if peer exists
	pubKey := p.ById[id].PubKeyHex

	delete(p.ByPubKey, pubKey)
	delete(p.ById, id)
}

/* ToSlice Methods */

func (p *Peers) ToPeerSlice() []*Peer {
	res := []*Peer{}

	for _, p := range p.ByPubKey {
		res = append(res, p)
	}

	sort.Sort(ByID(res))

	return res
}

func (p *Peers) ToPubKeySlice() []string {
	peers := p.ToPeerSlice()
	res := []string{}

	for _, peer := range peers {
		res = append(res, peer.PubKeyHex)
	}

	return res
}

func (p *Peers) ToIDSlice() []int {
	res := []int{}

	for id := range p.ById {
		res = append(res, id)
	}

	sort.Sort(ByInt(res))

	return res
}

/* Utilities */

func (p *Peers) Len() int {
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

type ByInt []int

func (a ByInt) Len() int      { return len(a) }
func (a ByInt) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByInt) Less(i, j int) bool {
	ai := a[i]
	aj := a[j]
	return ai < aj
}
