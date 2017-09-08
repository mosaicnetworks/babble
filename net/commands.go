package net

import "bitbucket.org/mosaicnet/babble/hashgraph"

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

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type FastForwardRequest struct {
	From string
}

type FastForwardResponse struct {
	From  string
	Head  string
	Seq   int
	Frame hashgraph.Frame
}
