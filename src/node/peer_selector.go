package node

import (
	"math/rand"

	"github.com/mosaicnetworks/babble/src/peers"
)

// peerSelector defines an interface for selecting the next gossip peer based
// on a list of peers.
type peerSelector interface {
	getPeers() *peers.PeerSet
	updateLast(peer uint32, connected bool) bool
	next() *peers.Peer
}

// randomPeerSelector implements the peerSelector interface and selects the next
// peer at random. It also keeps track of each peer's connection status.
type randomPeerSelector struct {
	peers                *peers.PeerSet
	selfID               uint32
	selectablePeersMap   map[uint32]*peerSelectorItem
	selectablePeersSlice []uint32
	last                 uint32
}

// peerSelectorItem is a wrapper around a Peer that keeps track of the
// connection status
type peerSelectorItem struct {
	peer      *peers.Peer
	connected bool
}

// newRandomPeerSelector creates a new randomPeerSelector from a PeerSet and
// excludes any peer identified by selfID.
func newRandomPeerSelector(peerSet *peers.PeerSet, selfID uint32) *randomPeerSelector {
	_, otherPeers := peers.ExcludePeer(peerSet.Peers, selfID)

	selectablePeersMap := map[uint32]*peerSelectorItem{}
	selectablePeersSlice := []uint32{}

	for _, p := range otherPeers {
		selectablePeersMap[p.ID()] = &peerSelectorItem{peer: p, connected: false}
		selectablePeersSlice = append(selectablePeersSlice, p.ID())
	}

	return &randomPeerSelector{
		peers:                peerSet,
		selfID:               selfID,
		selectablePeersMap:   selectablePeersMap,
		selectablePeersSlice: selectablePeersSlice,
	}
}

// getPeers returns the current set of peers.
func (ps *randomPeerSelector) getPeers() *peers.PeerSet {
	return ps.peers
}

// updateLast sets the last peer and updates its connection status.
func (ps *randomPeerSelector) updateLast(peer uint32, connected bool) (newConnection bool) {
	ps.last = peer

	// The peer could have been removed in by an InternalTransaction.
	if _, ok := ps.selectablePeersMap[peer]; ok {
		old := ps.selectablePeersMap[peer].connected

		ps.selectablePeersMap[peer].connected = connected

		if !old && connected {
			return true
		}
	}

	return false
}

// next returns the next peer.
func (ps *randomPeerSelector) next() *peers.Peer {

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

	peer := ps.selectablePeersMap[nextID].peer

	return peer
}
