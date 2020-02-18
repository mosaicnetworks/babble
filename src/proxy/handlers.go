package proxy

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
)

/*
These types are exported and need to be implemented and used by the calling
application.
*/

// ProxyHandler is used to instanciate an InmemProxy
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
