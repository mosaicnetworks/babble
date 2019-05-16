package proxy

import "github.com/mosaicnetworks/babble/src/hashgraph"

type CommitResponse struct {
	StateHash            []byte
	InternalTransactions []hashgraph.InternalTransaction
}

type CommitCallback func(block hashgraph.Block) (CommitResponse, error)

//DummyCommitCallback is used for testing
func DummyCommitCallback(block hashgraph.Block) (CommitResponse, error) {
	processedInternalTransactions := []hashgraph.InternalTransaction{}
	for _, it := range block.InternalTransactions() {
		pit := it.AsAccepted()
		processedInternalTransactions = append(processedInternalTransactions, pit)
	}

	response := CommitResponse{
		StateHash:            []byte{},
		InternalTransactions: processedInternalTransactions,
	}

	return response, nil
}
