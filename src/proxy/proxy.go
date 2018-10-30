package proxy

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
)

type AppProxy interface {
	SubmitCh() chan []byte
	SubmitInternalCh() chan *hashgraph.InternalTransaction
	CommitBlock(block hashgraph.Block) ([]byte, error)
	GetSnapshot(blockIndex int) ([]byte, error)
	Restore(snapshot []byte) error
}
