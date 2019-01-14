package node

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
)

type JoinPromiseResponse struct {
	AcceptedRound int
	Peers         []*peers.Peer
}

type JoinPromise struct {
	Tx     hashgraph.InternalTransaction
	RespCh chan JoinPromiseResponse
}

func NewJoinPromise(tx hashgraph.InternalTransaction) *JoinPromise {
	return &JoinPromise{
		Tx: tx,
		//make buffered because we don't want to block if there is no listener.
		//There might be something smarter to do here
		RespCh: make(chan JoinPromiseResponse, 2),
	}
}

func (p *JoinPromise) Respond(acceptedRound int, peers []*peers.Peer) {
	p.RespCh <- JoinPromiseResponse{acceptedRound, peers}
}
