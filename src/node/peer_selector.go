package node

import (
	"math/rand"

	"github.com/mosaicnetworks/babble/src/peers"
)

// PeerSelector defines and interface for Peer Selectors
type PeerSelector interface {
	Peers() *peers.PeerSet
	UpdateLast(peer uint32, connected bool) bool
	Next() *peers.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//RANDOM

// RandomPeerSelector defines a struct which controls the random selection of
// peers
type RandomPeerSelector struct {
	peers                *peers.PeerSet
	selfID               uint32
	selectablePeersMap   map[uint32]*PeerSelectorItem
	selectablePeersSlice []uint32
	last                 uint32
}

// PeerSelectorItem is a wrapper around a Peer that keeps track of the
// connection status
type PeerSelectorItem struct {
	Peer      *peers.Peer
	Connected bool
}

// NewRandomPeerSelector is a factory method that returns a new instance of
// RandomPeerSelector
func NewRandomPeerSelector(peerSet *peers.PeerSet, selfID uint32) *RandomPeerSelector {
	_, otherPeers := peers.ExcludePeer(peerSet.Peers, selfID)

	selectablePeersMap := map[uint32]*PeerSelectorItem{}
	selectablePeersSlice := []uint32{}

	for _, p := range otherPeers {
		selectablePeersMap[p.ID()] = &PeerSelectorItem{Peer: p, Connected: false}
		selectablePeersSlice = append(selectablePeersSlice, p.ID())
	}

	return &RandomPeerSelector{
		peers:                peerSet,
		selfID:               selfID,
		selectablePeersMap:   selectablePeersMap,
		selectablePeersSlice: selectablePeersSlice,
	}
}

//Peers returns the current set of peers
func (ps *RandomPeerSelector) Peers() *peers.PeerSet {
	return ps.peers
}

//UpdateLast sets the last peer and updates its connection status
func (ps *RandomPeerSelector) UpdateLast(peer uint32, connected bool) (newConnection bool) {
	ps.last = peer

	// The peer could have been removed in by an InternalTransaction
	if _, ok := ps.selectablePeersMap[peer]; ok {
		old := ps.selectablePeersMap[peer].Connected

		ps.selectablePeersMap[peer].Connected = connected

		if !old && connected {
			return true
		}
	}

	return false
}

//Next returns the next peer
func (ps *RandomPeerSelector) Next() *peers.Peer {

	if len(ps.selectablePeersSlice) == 0 {
		return nil
	}

	nextID := ps.selectablePeersSlice[0]

	if len(ps.selectablePeersSlice) > 1 {
		// remove last
		otherPeers := make([]uint32, 0, len(ps.selectablePeersSlice))
		for _, pid := range ps.selectablePeersSlice {
			if pid != ps.last {
				otherPeers = append(otherPeers, pid)
			}
		}

		// get random item from other peers
		nextID = otherPeers[rand.Intn(len(otherPeers))]
	}

	peer := ps.selectablePeersMap[nextID].Peer

	return peer
}
