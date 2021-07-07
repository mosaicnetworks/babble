package dummy

import (
	"fmt"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

// State represents the state of our dummy application. It implements the
// ProxyHandler interface for use with an InmemProxy. It doesn't really do
// anything useful but save and logs block transactions. The state hash is
// computed by cumulatively hashing transactions together as they come in.
// Snapshots correspond to the state hash resulting from executing a block's
// transactions.
type State struct {
	committedTxs [][]byte
	stateHash    []byte
	snapshots    map[int][]byte
	babbleState  state.State
	logger       *logrus.Entry
}

// NewState creates a new dummy state.
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

// CommitHandler implements the ProxyHandler interface. This callback is called
// Babble to commit a block to the application. Blocks contain transactions that
// represent commands or messages for the application, and internal transactions
// that are used internally by Babble to update the peer-set. The application
// can accept or refuse internal transactions based on custom rules. Here we
// accept all internal transactions. The commit response contains a state hash
// that represents the state of the application after applying all the
// transactions sequentially.
func (a *State) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	if a.logger.Level > logrus.InfoLevel {
		blockBytes, _ := block.Marshal()
		a.logger.WithField("block", string(blockBytes)).Debug("CommitBlock")
	}

	// Block transactions are ordered. Every Babble node will receive the same
	// transactions in the same order.
	a.committedTxs = append(a.committedTxs, block.Transactions()...)

	// The state hash is computed by hashing all transactions together.
	hash := a.stateHash
	for _, tx := range block.Transactions() {
		a.logger.Info(string(tx))
		hash = crypto.SimpleHashFromTwoHashes(hash, crypto.SHA256(tx))
	}
	a.stateHash = hash

	// Store the snapshot (which in the dummy application is the state hash) for
	// use by the SnapshotHandler.
	a.snapshots[block.Index()] = hash

	// Internal transactions represent requests to add or remove participants
	// from the Babble peer-set. This decision can be based on the application
	// state. For example the application could maintain a whitelist such that
	// only people whose public key belongs to the whitelist will be accepted to
	// join the peer-set. The decision must be deterministic, this is not a vote
	// where every one gives their opinion. All peers must return the same
	// answer, or risk creating a fork.
	receipts := []hashgraph.InternalTransactionReceipt{}
	for _, it := range block.InternalTransactions() {
		r := it.AsAccepted()
		receipts = append(receipts, r)
	}

	// The commit response contains the state-hash resulting from applying all
	// the transactions, and all the transaction receipts. This enables Babble
	// nodes to agree on the application state as well as the transaction log.
	// A block that is signed by a set of validators indicates that all these
	// validators agree that the application was in a given state after applying
	// the block. Accepted internal transaction receipts are used by Babble to
	// update the validator-set.
	response := proxy.CommitResponse{
		StateHash:                   a.stateHash,
		InternalTransactionReceipts: receipts,
	}

	return response, nil
}

// SnapshotHandler implements the ProxyHandler interface. It is used by Babble
// to retrieve a snapshot of the application at a specific block index. It is
// left to the application to keep track of snapshots and to encode/decode state
// snapshots to and from raw bytes. This handler is only used when fast-sync is
// activated.
func (a *State) SnapshotHandler(blockIndex int) ([]byte, error) {
	a.logger.WithField("block", blockIndex).Debug("GetSnapshot")

	snapshot, ok := a.snapshots[blockIndex]

	if !ok {
		return nil, fmt.Errorf("Snapshot %d not found", blockIndex)
	}

	return snapshot, nil
}

// RestoreHandler implements the ProxyHandler interface. It is called by Babble
// to instruct the application to restore its state back to a given snapshot.
// This is only used when fast-sync is activated.
func (a *State) RestoreHandler(snapshot []byte) ([]byte, error) {
	a.stateHash = snapshot

	return a.stateHash, nil
}

// StateChangeHandler implements the ProxyHandler interface. It is called by
// Babble to notify the application that the node has entered a new state (ex
// Babbling, Joining, Suspended, etc.).
func (a *State) StateChangeHandler(state state.State) error {
	a.babbleState = state
	a.logger.WithField("state", state).Debugf("StateChangeHandler")
	return nil
}

// GetCommittedTransactions returns the list of committed transactions
func (a *State) GetCommittedTransactions() [][]byte {
	return a.committedTxs
}
