package dummy

import (
	"fmt"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
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

type State struct {
	committedTxs [][]byte
	stateHash    []byte
	snapshots    map[int][]byte
	logger       *logrus.Logger
}

func NewState(logger *logrus.Logger) *State {
	state := &State{
		committedTxs: [][]byte{},
		stateHash:    []byte{},
		snapshots:    make(map[int][]byte),
		logger:       logger,
	}
	logger.Info("Init Dummy State")
	return state
}

func (a *State) CommitHandler(block hashgraph.Block) ([]byte, error) {
	a.logger.WithField("block", block).Debug("CommitBlock")
	err := a.commit(block)
	if err != nil {
		return nil, err
	}
	return a.stateHash, nil
}

func (a *State) SnapshotHandler(blockIndex int) ([]byte, error) {
	a.logger.WithField("block", blockIndex).Debug("GetSnapshot")

	snapshot, ok := a.snapshots[blockIndex]
	if !ok {
		return nil, fmt.Errorf("Snapshot %d not found", blockIndex)
	}

	return snapshot, nil
}

func (a *State) RestoreHandler(snapshot []byte) ([]byte, error) {
	//XXX do something smart here
	a.stateHash = snapshot
	return a.stateHash, nil
}

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

	//XXX do something smart here
	a.snapshots[block.Index()] = hash

	return nil
}
