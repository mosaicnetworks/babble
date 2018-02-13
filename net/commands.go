package net

import "github.com/babbleio/babble/hashgraph"

type SyncRequest struct {
	FromID               int
	KnownEvents          map[int]int
	KnownBlockSignatures map[int]int
}

type SyncResponse struct {
	FromID               int
	SyncLimit            bool
	Events               []hashgraph.WireEvent
	BlockSignatures      []hashgraph.BlockSignature
	KnownEvents          map[int]int
	KnownBlockSignatures map[int]int
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type EagerSyncRequest struct {
	FromID          int
	Events          []hashgraph.WireEvent
	BlockSignatures []hashgraph.BlockSignature
}

type EagerSyncResponse struct {
	FromID  int
	Success bool
}
