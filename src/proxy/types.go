package proxy

import "github.com/mosaicnetworks/babble/src/hashgraph"

type CommitResponse struct {
	StateHash            []byte
	InternalTransactions []hashgraph.InternalTransaction
}

type CommitCallback func(block hashgraph.Block) (CommitResponse, error)

//DummyCommitCallback is used for testing
func DummyCommitCallback(block hashgraph.Block) (CommitResponse, error) {
	for _, it := range block.InternalTransactions() {
		it.Accept()
	}

	res := CommitResponse{
		StateHash:            []byte{},
		InternalTransactions: block.InternalTransactions(),
	}

	return res, nil
}
