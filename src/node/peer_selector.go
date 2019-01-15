package node

import (
	"math/rand"

	"github.com/mosaicnetworks/babble/src/peers"
)

type PeerSelector interface {
	Peers() *peers.PeerSet
	UpdateLast(peer uint32)
	Next() *peers.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//RANDOM

type RandomPeerSelector struct {
	peers           *peers.PeerSet
	selfID          uint32
	selectablePeers []*peers.Peer
	last            uint32
}

func NewRandomPeerSelector(peerSet *peers.PeerSet, selfID uint32) *RandomPeerSelector {
	_, selectablePeers := peers.ExcludePeer(peerSet.Peers, selfID)
	return &RandomPeerSelector{
		peers:           peerSet,
		selfID:          selfID,
		selectablePeers: selectablePeers,
	}
}

func (ps *RandomPeerSelector) Peers() *peers.PeerSet {
	return ps.peers
}

func (ps *RandomPeerSelector) UpdateLast(peer uint32) {
	ps.last = peer
}

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
