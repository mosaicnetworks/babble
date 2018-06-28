package proxy

import "github.com/mosaicnetworks/babble/hashgraph"

type AppProxy interface {
	SubmitCh() chan []byte
	CommitBlock(block hashgraph.Block) ([]byte, error)
	GetSnapshot(blockIndex int) ([]byte, error)
	Restore(snapshot []byte) error
}

//XXX This interface is never used... maybe in SocketBabbleProxy
type BabbleProxy interface {
	CommitCh() chan hashgraph.Block
	SnapshotRequestCh() chan int
	RestoreCh() chan []byte
	SubmitTx(tx []byte) error
}
