package net

import "bitbucket.org/mosaicnet/babble/hashgraph"

type SyncRequest struct {
	From  string
	Known map[int]int
}

type SyncResponse struct {
	From   string
	Events []hashgraph.WireEvent
	Known  map[int]int
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
