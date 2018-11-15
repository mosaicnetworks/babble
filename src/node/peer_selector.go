package node

import (
	"math/rand"

	"github.com/mosaicnetworks/babble/src/peers"
)

//XXX PeerSelector needs major refactoring

type PeerSelector interface {
	Peers() *peers.PeerSet
	UpdateLast(peer int)
	Next() *peers.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//RANDOM

type RandomPeerSelector struct {
	peers  *peers.PeerSet
	selfID int
	last   int
}

func NewRandomPeerSelector(peerSet *peers.PeerSet, selfID int) *RandomPeerSelector {
	return &RandomPeerSelector{
		selfID: selfID,
		peers:  peerSet,
	}
}

func (ps *RandomPeerSelector) Peers() *peers.PeerSet {
	return ps.peers
}

func (ps *RandomPeerSelector) UpdateLast(peer int) {
	ps.last = peer
}

func (ps *RandomPeerSelector) Next() *peers.Peer {
	selectablePeers := ps.peers.Peers

	if len(selectablePeers) > 1 {
		_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.selfID)

		if len(selectablePeers) > 1 {
			_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.last)
		}
	}

	i := rand.Intn(len(selectablePeers))

	peer := selectablePeers[i]

	return peer
}
