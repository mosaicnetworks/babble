package inmem

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/proxy"
)

// ExampleHandler implements the ProxyHandler interface. This is where an
// application would normally register callbacks that Babble will call through
// the InmemProxy. ExampleHandler simply maintains a list of all the committed
// transactions in the order they were received from Babble, and keeps track of
// Babble's state. Refer to the dummy package for a more meaningful example.
type ExampleHandler struct {
	transactions [][]byte
	state        state.State
}

// CommitHandler is called by Babble to commit a block to the application.
// Blocks contain transactions that represent commands for the application, and
// internal transactions that are used internally by Babble to update the
// peer-set. The application can accept or refuse internal transactions based on
// custom rules. Here we accept all internal transactions. The commit response
// contains a state hash that should represent the state of the application
// after applying all the transactions sequentially.
func (p *ExampleHandler) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	// block transactions are ordered. Every Babble node will receive the same
	// transactions in the same order.
	p.transactions = append(p.transactions, block.Transactions()...)

	// internal transactions represent requests to add or remove participants
	// from the Babble peer-set. This decision can be based on the application
	// state. For example the application could maintain a whitelist such that
	// only people whose public key belongs to the whitelist will be accepted to
	// join the peer-set. The decision must be deterministic, this is not a vote
	// where every one gives their opinion. All peers must return the same
	// answer, or risk creating a fork.
	receipts := []hashgraph.InternalTransactionReceipt{}
	for _, it := range block.InternalTransactions() {
		receipts = append(receipts, it.AsAccepted())
	}

	// The commit response contains the state-hash resulting from applying all
	// the transactions, and all the transaction receipts. Here we always
	// return the same hard-coded state-hash.
	response := proxy.CommitResponse{
		StateHash:                   []byte("statehash"),
		InternalTransactionReceipts: receipts,
	}

	return response, nil
}

// StateChangedHandler is called by Babble to notify the application that the
// node has entered a new state (ex Babbling, Joining, Suspended, etc.).
func (p *ExampleHandler) StateChangeHandler(state state.State) error {
	p.state = state
	return nil
}

// SnapshotHandler is used by Babble to retrieve a snapshot of the application
// corresponding to a specific block index. It is left to the application to
// keep track of snapshots and to encode/decode state snapshots to and from raw
// bytes. This handler is only used when fast-sync is activated.
func (p *ExampleHandler) SnapshotHandler(blockIndex int) ([]byte, error) {
	return []byte("snapshot"), nil
}

// RestoreHandler is called by Babble to instruct the application to restore its
// state back to a given snapshot. This is only used when fast-sync is
// activated.
func (p *ExampleHandler) RestoreHandler(snapshot []byte) ([]byte, error) {
	return []byte("statehash"), nil
}

func NewExampleHandler() *ExampleHandler {
	return &ExampleHandler{
		transactions: [][]byte{},
	}
}

func Example() {
	// An application needs to implement the ProxyHandler interface and define
	// the callbacks that will be automatically called by the proxy when Babble
	// has things to communicate to the application.
	handler := NewExampleHandler()

	// We create an InmemProxy based on the handler. Normally the InmemProxy is
	// passed to a Babble node but this example only illustrates the creation of
	// an InmemProxy.
	proxy := NewInmemProxy(handler, nil)

	// The application can submit transactions to Babble using the proxy's
	// SubmitTx. Babble will broadcast the transactions to other nodes, run
	// them through the consensus algorithm, and eventually call the callback
	// methods implemented in the handler.
	proxy.SubmitTx([]byte("the test transaction"))
}
