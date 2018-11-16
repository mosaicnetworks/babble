package proxy

import "github.com/mosaicnetworks/babble/src/hashgraph"

type CommitResponse struct {
	StateHash                    []byte
	AcceptedInternalTransactions []hashgraph.InternalTransaction
}

type CommitCallback func(block hashgraph.Block) (CommitResponse, error)

//DummyCommitCallback is used for testing
func DummyCommitCallback(block hashgraph.Block) (CommitResponse, error) {
	acceptedInternalTransactions := make([]bool, len(block.InternalTransactions()))
	for i := range block.InternalTransactions() {
		acceptedInternalTransactions[i] = true
	}

	res := CommitResponse{
		StateHash:                    []byte{},
		AcceptedInternalTransactions: block.InternalTransactions(),
	}

	return res, nil
}
