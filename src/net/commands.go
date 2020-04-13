package net

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
)

// SyncRequest corresponds to  the pull part of the pull-push gossip protocol.
// It is used to retrieve unknown Events from another node. The Known map
// represents how much the requester currently knows about the hashgraph. The
// SyncLimit indicates the max number of Events to include in the response.
type SyncRequest struct {
	FromID    uint32
	Known     map[uint32]int
	SyncLimit int
}

// SyncResponse returns a list of Events as requested by a SyncRequest. The
// known map indicates how much the responder knows about the hashgraph. Events
// are encoded in light-weight wire format to take less space.
type SyncResponse struct {
	FromID uint32
	Events []hashgraph.WireEvent
	Known  map[uint32]int
}

// EagerSyncRequest corresponds to the push part of the pull-push gossip
// protocol. It is used to actively push Events to a node without it being
// requested.
type EagerSyncRequest struct {
	FromID uint32
	Events []hashgraph.WireEvent
}

// EagerSyncResponse indicates the success or failure of an EagerSyncRequest.
type EagerSyncResponse struct {
	FromID  uint32
	Success bool
}

// FastForwardRequest is used to request a Block, Frame, and Snapshot, from
// which to fast-forward.
type FastForwardRequest struct {
	FromID uint32
}

// FastForwardResponse encapsulates the response to a FastForwardRequest.
type FastForwardResponse struct {
	FromID   uint32
	Block    hashgraph.Block
	Frame    hashgraph.Frame
	Snapshot []byte
}

// JoinRequest is used to submit an InternalTransaction to join a Babble group.
type JoinRequest struct {
	InternalTransaction hashgraph.InternalTransaction
}

// JoinResponse contains the response to a JoinRequest.
type JoinResponse struct {
	FromID        uint32
	Accepted      bool
	AcceptedRound int
	Peers         []*peers.Peer
}
