package proxy

import (
	"github.com/mosaicnetworks/babble/hashgraph"
	bproxy "github.com/mosaicnetworks/babble/proxy/babble"
)

type AppProxy interface {
	SubmitCh() chan []byte
	CommitBlock(block hashgraph.Block) ([]byte, error)
	GetSnapshot(blockIndex int) ([]byte, error)
	Restore(snapshot []byte) error
}

type BabbleProxy interface {
	CommitCh() chan bproxy.Commit
	SnapshotRequestCh() chan bproxy.SnapshotRequest
	RestoreCh() chan bproxy.RestoreRequest
	SubmitTx(tx []byte) error
}
