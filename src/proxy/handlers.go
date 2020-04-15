package proxy

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
)

// ProxyHandler encapsulates callbacks to be called by the InmemProxy. This is
// the true contact surface between Babble and the Application. The application
// must implement these handlers to process incoming Babble blocks and state
// changes. The Snapshot and Restore handlers are necessary only with fast-sync,
// which is still experimental, so can safely be empty.
type ProxyHandler interface {
	// CommitHandler is called when Babble commits a block to the application
	CommitHandler(block hashgraph.Block) (response CommitResponse, err error)

	// SnapshotHandler is called by Babble to retrieve a snapshot corresponding
	// to a particular block
	SnapshotHandler(blockIndex int) (snapshot []byte, err error)

	// RestoreHandler is called by Babble to restore the application to a
	// specific state
	RestoreHandler(snapshot []byte) (stateHash []byte, err error)

	// StateChangeHandler is called by onStateChanged to notify that a Babble
	// node entered a certain state
	StateChangeHandler(state.State) error
}
