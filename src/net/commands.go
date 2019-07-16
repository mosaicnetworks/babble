package net

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
)

// SyncRequest ...
type SyncRequest struct {
	FromID    uint32
	Known     map[uint32]int
	SyncLimit int
}

// SyncResponse ...
type SyncResponse struct {
	FromID uint32
	Events []hashgraph.WireEvent
	Known  map[uint32]int
}

// EagerSyncRequest ...
type EagerSyncRequest struct {
	FromID uint32
	Events []hashgraph.WireEvent
}

// EagerSyncResponse ...
type EagerSyncResponse struct {
	FromID  uint32
	Success bool
}

// FastForwardRequest ...
type FastForwardRequest struct {
	FromID uint32
}

// FastForwardResponse ...
type FastForwardResponse struct {
	FromID   uint32
	Block    hashgraph.Block
	Frame    hashgraph.Frame
	Snapshot []byte
}

// JoinRequest ...
type JoinRequest struct {
	InternalTransaction hashgraph.InternalTransaction
}

// JoinResponse ...
type JoinResponse struct {
	FromID        uint32
	Accepted      bool
	AcceptedRound int
	Peers         []*peers.Peer
}
