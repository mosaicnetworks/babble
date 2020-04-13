package node

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
)

// joinPromiseResponse is a struct returned by a joinPromise.
type joinPromiseResponse struct {
	accepted      bool
	acceptedRound int
	peers         []*peers.Peer
}

// joinPromise is a relay between the requestJoin RPC handler (which receives
// join requests from other peers), and the consensus system which
// asynchronously processes the corresponding InternalTransaction. It is also
// used with leave requests.
type joinPromise struct {
	tx     hashgraph.InternalTransaction
	respCh chan joinPromiseResponse
}

// newJoinPromise is a factory method for a joinPromise
func newJoinPromise(tx hashgraph.InternalTransaction) *joinPromise {
	return &joinPromise{
		tx: tx,
		// buffered because we don't want to block if there is no listener.
		// There might be something smarter to do here
		respCh: make(chan joinPromiseResponse, 2),
	}
}

// respond handles sending a joinPromiseResponse to a joinPromise
func (p *joinPromise) respond(accepted bool, acceptedRound int, peers []*peers.Peer) {
	p.respCh <- joinPromiseResponse{accepted, acceptedRound, peers}
}
