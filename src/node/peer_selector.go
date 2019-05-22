package node

import (
	"math/rand"

	"github.com/mosaicnetworks/babble/src/peers"
)

//PeerSelector defines and interface for Peer Selectors
type PeerSelector interface {
	Peers() *peers.PeerSet
	UpdateLast(peer uint32)
	Next() *peers.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//RANDOM

//RandomPeerSelector defines a struct which controls the random selection of peers
type RandomPeerSelector struct {
	peers           *peers.PeerSet
	selfID          uint32
	selectablePeers []*peers.Peer
	last            uint32
}

//NewRandomPeerSelector is a factory method that returns a new instance of RandomPeerSelector
func NewRandomPeerSelector(peerSet *peers.PeerSet, selfID uint32) *RandomPeerSelector {
	_, selectablePeers := peers.ExcludePeer(peerSet.Peers, selfID)
	return &RandomPeerSelector{
		peers:           peerSet,
		selfID:          selfID,
		selectablePeers: selectablePeers,
	}
}

//Peers returns a set of peers
func (ps *RandomPeerSelector) Peers() *peers.PeerSet {
	return ps.peers
}

//UpdateLast sets the last peer
func (ps *RandomPeerSelector) UpdateLast(peer uint32) {
	ps.last = peer
}

//Next returns the next peer
func (ps *RandomPeerSelector) Next() *peers.Peer {
	selectablePeers := ps.selectablePeers

	if len(selectablePeers) == 0 {
		return nil
	}

	if len(selectablePeers) > 1 {
		_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.last)
	}

	i := rand.Intn(len(selectablePeers))

	peer := selectablePeers[i]

	return peer
}
