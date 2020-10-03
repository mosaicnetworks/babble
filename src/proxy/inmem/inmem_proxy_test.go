package inmem

import (
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestInmemProxyAppSide(t *testing.T) {
	proxy := NewInmemProxy(
		NewExampleHandler(),
		common.NewTestEntry(t, common.TestLogLevel),
	)

	submitCh := proxy.SubmitCh()

	tx := []byte("the test transaction")

	// Listen for a request
	go func() {
		select {
		case st := <-submitCh:
			// Verify the command
			if !reflect.DeepEqual(st, tx) {
				t.Fatalf("tx mismatch: %#v %#v", tx, st)
			}

		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout")
		}
	}()

	proxy.SubmitTx(tx)
}

func TestInmemProxyBabbleSide(t *testing.T) {
	handler := NewExampleHandler()

	proxy := NewInmemProxy(
		handler,
		common.NewTestEntry(t, common.TestLogLevel),
	)

	transactions := [][]byte{
		[]byte("tx 1"),
		[]byte("tx 2"),
		[]byte("tx 3"),
	}

	block := hashgraph.NewBlock(0, 1, []byte{}, []*peers.Peer{}, transactions, []hashgraph.InternalTransaction{}, 0)

	/***************************************************************************
	Commit
	***************************************************************************/
	commitResponse, err := proxy.CommitBlock(*block)
	if err != nil {
		t.Fatal(err)
	}

	expectedStateHash := []byte("statehash")
	if !reflect.DeepEqual(commitResponse.StateHash, expectedStateHash) {
		t.Fatalf("StateHash should be %v, not %v", expectedStateHash, commitResponse.StateHash)
	}

	if !reflect.DeepEqual(transactions, handler.transactions) {
		t.Fatalf("Transactions should be %v, not %v", transactions, handler.transactions)
	}

	/***************************************************************************
	Snapshot
	***************************************************************************/
	snapshot, err := proxy.GetSnapshot(block.Index())
	if err != nil {
		t.Fatal(err)
	}

	expectedSnapshot := []byte("snapshot")
	if !reflect.DeepEqual(snapshot, expectedSnapshot) {
		t.Fatalf("Snapshot should be %v, not %v", expectedSnapshot, snapshot)
	}

	/***************************************************************************
	Restore
	***************************************************************************/

	err = proxy.Restore(snapshot)
	if err != nil {
		t.Fatalf("Error restoring snapshot: %v", err)
	}

	/***************************************************************************
	State
	***************************************************************************/
	err = proxy.OnStateChanged(state.Babbling)
	if err != nil {
		t.Fatal(err)
	}

	if handler.state != state.Babbling {
		t.Fatalf("Proxy state should be Babbling, not %v", handler.state.String())
	}

}
