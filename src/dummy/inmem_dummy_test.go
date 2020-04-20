package dummy

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/babble"
	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/config"
	bcrypto "github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestInmemDummyAppSide(t *testing.T) {
	logger := common.NewTestEntry(t, common.TestLogLevel)

	dummy := NewInmemDummyClient(logger)

	tx := []byte("the test transaction")

	go func() {
		select {
		case st := <-dummy.SubmitCh():
			// Verify the command
			if !reflect.DeepEqual(st, tx) {
				t.Fatalf("tx mismatch: %#v %#v", tx, st)
			}

		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout")
		}
	}()

	dummy.SubmitTx(tx)
}

func TestInmemDummyServerSide(t *testing.T) {
	logger := common.NewTestEntry(t, common.TestLogLevel)

	dummy := NewInmemDummyClient(logger)

	//create a few blocks
	blocks := [5]*hashgraph.Block{}

	for i := 0; i < 5; i++ {
		blocks[i] = hashgraph.NewBlock(i, i+1,
			[]byte{},
			[]*peers.Peer{},
			[][]byte{
				[]byte(fmt.Sprintf("block %d transaction", i)),
			},
			[]hashgraph.InternalTransaction{
				hashgraph.NewInternalTransaction(hashgraph.PEER_ADD, *peers.NewPeer("node0", "paris", "")),
				hashgraph.NewInternalTransaction(hashgraph.PEER_REMOVE, *peers.NewPeer("node1", "london", "")),
			},
		)
	}

	//commit first block and check that the client's statehash is correct
	commitResponse, err := dummy.CommitBlock(*blocks[0])

	if err != nil {
		t.Fatalf("Fatal Error: %v", err)
	}

	expectedStateHash := []byte{}

	for _, t := range blocks[0].Transactions() {
		tHash := bcrypto.SHA256(t)

		expectedStateHash = bcrypto.SimpleHashFromTwoHashes(expectedStateHash, tHash)
	}

	if !reflect.DeepEqual(commitResponse.StateHash, expectedStateHash) {
		t.Fatalf("StateHash should be %v, not %v", expectedStateHash, commitResponse.StateHash)
	}

	snapshot, err := dummy.GetSnapshot(blocks[0].Index())

	if err != nil {
		t.Fatalf("Fatal Error: %v", err)
	}

	if !reflect.DeepEqual(snapshot, expectedStateHash) {
		t.Fatalf("Snapshot should be %v, not %v", expectedStateHash, snapshot)
	}

	//commit a few more blocks, then attempt to restore back to block 0 state
	for i := 1; i < 5; i++ {
		_, err := dummy.CommitBlock(*blocks[i])

		if err != nil {
			t.Fatalf("Fatal Error: %v", err)
		}
	}

	err = dummy.Restore(snapshot)

	if err != nil {
		t.Fatalf("Error restoring snapshot: %v", err)
	}

	if !reflect.DeepEqual(dummy.state.stateHash, expectedStateHash) {
		t.Fatalf("Restore StateHash should be %v, not %v", expectedStateHash, dummy.state.stateHash)
	}

	err = dummy.OnStateChanged(state.Babbling)

	if err != nil {
		t.Fatalf("Error in OnStateChanged: %v", err)
	}

	if !reflect.DeepEqual(dummy.state.babbleState, state.Babbling) {
		t.Fatalf("Babble State should be Babbling, not %v", dummy.state.babbleState.String())
	}
}

func ExampleInmemDummyClient() {
	// Start from default Babble configuration.
	babbleConfig := config.NewDefaultConfig()

	// Create dummy InmemProxy
	dummy := NewInmemDummyClient(babbleConfig.Logger())

	// Set the proxy in the Babble configuration.
	babbleConfig.Proxy = dummy

	// Instantiate Babble.
	babble := babble.NewBabble(babbleConfig)

	// Read in the configuration and initialise the node accordingly.
	if err := babble.Init(); err != nil {
		babbleConfig.Logger().Error("Cannot initialize babble:", err)
		os.Exit(1)
	}

	// The application can submit transactions to Babble using the proxy's
	// SubmitTx. Babble will broadcast the transactions to other nodes, run them
	// through the consensus algorithm, and eventually call the callback methods
	// implemented in the handler.
	go func() {
		dummy.SubmitTx([]byte("the test transaction"))
	}()

	// Run the node aynchronously.
	babble.Run()

	// Babble reacts to SIGINT (Ctrl + c) and SIGTERM by calling the leave
	// method to politely leave a Babble network, but it can also be called
	// manually.
	defer babble.Node.Leave()
}
