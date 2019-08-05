package dummy

import (
	"fmt"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

/*
* The dummy App is used for testing and as an example for building Babble
* applications. Here, we define the dummy's state which doesn't really do
* anything useful. It saves and logs block transactions. The state hash is
* computed by cumulatively hashing transactions together as they come in.
* Snapshots correspond to the state hash resulting from executing a the block's
* transactions.
 */

// State ...
type State struct {
	committedTxs [][]byte
	stateHash    []byte
	snapshots    map[int][]byte
	logger       *logrus.Entry
}

// NewState ...
func NewState(logger *logrus.Entry) *State {
	state := &State{
		committedTxs: [][]byte{},
		stateHash:    []byte{},
		snapshots:    make(map[int][]byte),
		logger:       logger,
	}

	logger.Info("Init Dummy State")

	return state
}

// CommitHandler ...
func (a *State) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	a.logger.WithField("block", block).Debug("CommitBlock")

	err := a.commit(block)
	if err != nil {
		return proxy.CommitResponse{}, err
	}

	receipts := []hashgraph.InternalTransactionReceipt{}
	for _, it := range block.InternalTransactions() {
		r := it.AsAccepted()
		receipts = append(receipts, r)
	}

	response := proxy.CommitResponse{
		StateHash:                   a.stateHash,
		InternalTransactionReceipts: receipts,
	}

	return response, nil
}

// SnapshotHandler ...
func (a *State) SnapshotHandler(blockIndex int) ([]byte, error) {
	a.logger.WithField("block", blockIndex).Debug("GetSnapshot")

	snapshot, ok := a.snapshots[blockIndex]

	if !ok {
		return nil, fmt.Errorf("Snapshot %d not found", blockIndex)
	}

	return snapshot, nil
}

// RestoreHandler ...
func (a *State) RestoreHandler(snapshot []byte) ([]byte, error) {
	a.stateHash = snapshot

	return a.stateHash, nil
}

// GetCommittedTransactions ...
func (a *State) GetCommittedTransactions() [][]byte {
	return a.committedTxs
}

func (a *State) commit(block hashgraph.Block) error {
	a.committedTxs = append(a.committedTxs, block.Transactions()...)

	//log tx and update state hash
	hash := a.stateHash

	for _, tx := range block.Transactions() {
		a.logger.Info(string(tx))

		hash = crypto.SimpleHashFromTwoHashes(hash, crypto.SHA256(tx))
	}

	a.stateHash = hash

	a.snapshots[block.Index()] = hash

	return nil
}
