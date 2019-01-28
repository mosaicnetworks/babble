package proxy

import "github.com/mosaicnetworks/babble/src/hashgraph"

type CommitResponse struct {
	StateHash []byte
}

type CommitCallback func(block hashgraph.Block) (CommitResponse, error)

//DummyCommitCallback is used for testing
func DummyCommitCallback(block hashgraph.Block) (CommitResponse, error) {
	res := CommitResponse{
		StateHash: []byte{},
	}

	return res, nil
}
