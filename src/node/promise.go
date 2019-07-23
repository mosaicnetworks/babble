package node

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*

JoinPromise acts as a relay between the requestJoin RPC handler (which receives
join requests from other peers), and the consensus system which asynchronously
processes the corresponding InternalTransaction. It is also used with leave
requests.

*/

// JoinPromiseResponse is a struct returned by a JoinPromise
type JoinPromiseResponse struct {
	Accepted      bool
	AcceptedRound int
	Peers         []*peers.Peer
}

// JoinPromise is a struct for an asynchronous response to a Join Request
type JoinPromise struct {
	Tx     hashgraph.InternalTransaction
	RespCh chan JoinPromiseResponse
}

// NewJoinPromise is a factory method for a JoinPromise
func NewJoinPromise(tx hashgraph.InternalTransaction) *JoinPromise {
	return &JoinPromise{
		Tx: tx,
		// buffered because we don't want to block if there is no listener.
		// There might be something smarter to do here
		RespCh: make(chan JoinPromiseResponse, 2),
	}
}

// Respond handles sending a JoinPromiseResponse to a JoinPromise
func (p *JoinPromise) Respond(accepted bool, acceptedRound int, peers []*peers.Peer) {
	p.RespCh <- JoinPromiseResponse{accepted, acceptedRound, peers}
}
