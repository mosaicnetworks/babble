package node

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
)

type JoinPromise struct {
	Tx     hashgraph.InternalTransaction
	RespCh chan bool
	hash   string
}

func NewJoinPromise(tx hashgraph.InternalTransaction) *JoinPromise {
	return &JoinPromise{
		Tx: tx,
		//XXX make buffered because we don't want to block if there is no
		//listener. There might be something smarter to do here
		RespCh: make(chan bool, 2),
	}
}

func (p *JoinPromise) Respond(res bool) {
	p.RespCh <- res
}
