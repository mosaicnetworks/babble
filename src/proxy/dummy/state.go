package dummy

import (
	"fmt"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

/*
* The dummy App is used for testing and as an example for building Babble
* applications. Here, we define the dummy's state which doesn't really do
* anything useful. It saves and logs block transactions. The state hash is
* computed by cumulatively hashing transactions together as they come in.
* Snapshots correspond to the state hash resulting from executing a block's
* transactions.
 */

// State represents the state of our dummy application
type State struct {
	committedTxs [][]byte
	stateHash    []byte
	snapshots    map[int][]byte
	babbleState  state.State
	logger       *logrus.Entry
}

// NewState creates a new state
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

// CommitHandler implements the ProxyHandler interface
func (a *State) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {

	if a.logger.Level > logrus.InfoLevel {
		blockBytes, _ := block.Marshal()
		a.logger.WithField("block", string(blockBytes)).Debug("CommitBlock")
	}

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

// SnapshotHandler implements the ProxyHandler interface
func (a *State) SnapshotHandler(blockIndex int) ([]byte, error) {
	a.logger.WithField("block", blockIndex).Debug("GetSnapshot")

	snapshot, ok := a.snapshots[blockIndex]

	if !ok {
		return nil, fmt.Errorf("Snapshot %d not found", blockIndex)
	}

	return snapshot, nil
}

// RestoreHandler implements the ProxyHandler interface
func (a *State) RestoreHandler(snapshot []byte) ([]byte, error) {
	a.stateHash = snapshot

	return a.stateHash, nil
}

// StateChangeHandler implements the ProxyHandler interface
func (a *State) StateChangeHandler(state state.State) error {
	a.babbleState = state

	a.logger.WithField("state", state).Debugf("StateChangeHandler")

	return nil
}

// GetCommittedTransactions returns the list of committed transactions
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
