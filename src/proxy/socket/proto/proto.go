package proto

import "github.com/mosaicnetworks/babble/src/hashgraph"

type StateHash struct {
	Hash []byte
}

// CommitResponse captures both a response and a potential error.
type CommitResponse struct {
	StateHash []byte
	Error     error
}

// Commit provides a response mechanism.
type Commit struct {
	Block    hashgraph.Block
	RespChan chan<- CommitResponse
}

// Respond is used to respond with a response, error or both
func (r *Commit) Respond(stateHash []byte, err error) {
	r.RespChan <- CommitResponse{stateHash, err}
}

//------------------------------------------------------------------------------

type Snapshot struct {
	Bytes []byte
}

// SnapshotResponse captures both a response and a potential error.
type SnapshotResponse struct {
	Snapshot []byte
	Error    error
}

// SnapshotRequest provides a response mechanism.
type SnapshotRequest struct {
	BlockIndex int
	RespChan   chan<- SnapshotResponse
}

// Respond is used to respond with a response, error or both
func (r *SnapshotRequest) Respond(snapshot []byte, err error) {
	r.RespChan <- SnapshotResponse{snapshot, err}
}

//------------------------------------------------------------------------------

// RestoreResponse captures both an error.
type RestoreResponse struct {
	StateHash []byte
	Error     error
}

// RestoreRequest provides a response mechanism.
type RestoreRequest struct {
	Snapshot []byte
	RespChan chan<- RestoreResponse
}

// Respond is used to respond with a response, error or both
func (r *RestoreRequest) Respond(snapshot []byte, err error) {
	r.RespChan <- RestoreResponse{snapshot, err}
}
