package proxy

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
)

// AppProxy defines the interface which is used by Babble to communicate with
// the App
type AppProxy interface {
	SubmitCh() chan []byte
	CommitBlock(block hashgraph.Block) (CommitResponse, error)
	GetSnapshot(blockIndex int) ([]byte, error)
	Restore(snapshot []byte) error
}
