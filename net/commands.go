package net

import "github.com/babbleio/babble/hashgraph"

type SyncRequest struct {
	From  string
	Known map[int]int
}

type SyncResponse struct {
	From      string
	SyncLimit bool
	Events    []hashgraph.WireEvent
	Known     map[int]int
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type EagerSyncRequest struct {
	From   string
	Events []hashgraph.WireEvent
}

type EagerSyncResponse struct {
	From    string
	Success bool
}
