package net

import "github.com/mosaicnetworks/babble/hashgraph"

type SyncRequest struct {
	FromID int
	Known  map[int]int
}

type SyncResponse struct {
	FromID    int
	SyncLimit bool
	Events    []hashgraph.WireEvent
	Known     map[int]int
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type EagerSyncRequest struct {
	FromID int
	Events []hashgraph.WireEvent
}

type EagerSyncResponse struct {
	FromID  int
	Success bool
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type FastForwardRequest struct {
	FromID int
}

type FastForwardResponse struct {
	FromID   int
	Block    hashgraph.Block
	Frame    hashgraph.Frame
	Snapshot []byte
}
