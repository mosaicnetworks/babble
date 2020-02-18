package proxy

import "github.com/mosaicnetworks/babble/src/hashgraph"

// CommitResponse ...
type CommitResponse struct {
	StateHash                   []byte
	InternalTransactionReceipts []hashgraph.InternalTransactionReceipt
}

// CommitCallback ...
type CommitCallback func(block hashgraph.Block) (CommitResponse, error)

// DummyCommitCallback is used for testing
func DummyCommitCallback(block hashgraph.Block) (CommitResponse, error) {
	receipts := []hashgraph.InternalTransactionReceipt{}
	for _, it := range block.InternalTransactions() {
		r := it.AsAccepted()
		receipts = append(receipts, r)
	}

	response := CommitResponse{
		StateHash:                   []byte{},
		InternalTransactionReceipts: receipts,
	}

	return response, nil
}
