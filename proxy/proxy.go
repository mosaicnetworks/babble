package proxy

import "github.com/mosaicnetworks/babble/hashgraph"

type AppProxy interface {
	SubmitCh() chan []byte
	CommitBlock(block hashgraph.Block) ([]byte, error)
}

type BabbleProxy interface {
	CommitCh() chan hashgraph.Block
	SubmitTx(tx []byte) error
}
