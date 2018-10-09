package socket

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	bcrypto "github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/dummy/state"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/socket/app"
	bproxy "github.com/mosaicnetworks/babble/src/proxy/socket/babble"
)

func TestSocketProxyServer(t *testing.T) {
	clientAddr := "127.0.0.1:9990"
	proxyAddr := "127.0.0.1:9991"

	appProxy, err := aproxy.NewSocketAppProxy(clientAddr, proxyAddr, 1*time.Second, common.NewTestLogger(t))

	if err != nil {
		t.Fatalf("Cannot create SocketAppProxy: %s", err)
	}

	submitCh := appProxy.SubmitCh()

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

	// now client part connecting to RPC service
	// and calling methods
	babbleProxy, err := bproxy.NewSocketBabbleProxy(proxyAddr, clientAddr, 1*time.Second, common.NewTestLogger(t))

	if err != nil {
		t.Fatal(err)
	}

	err = babbleProxy.SubmitTx(tx)

	if err != nil {
		t.Fatal(err)
	}
}

func TestSocketProxyClient(t *testing.T) {
	clientAddr := "127.0.0.1:9992"
	proxyAddr := "127.0.0.1:9993"

	logger := common.NewTestLogger(t)

	//create app proxy
	appProxy, err := aproxy.NewSocketAppProxy(clientAddr, proxyAddr, 1*time.Second, logger)
	if err != nil {
		t.Fatalf("Cannot create SocketAppProxy: %s", err)
	}

	//create babble proxy
	babbleProxy, err := bproxy.NewSocketBabbleProxy(proxyAddr, clientAddr, 1*time.Second, logger)

	state := state.NewState(logger)

	initialStateHash := []byte{}

	go func() {
		for {
			select {
			case commit := <-babbleProxy.CommitCh():
				t.Log("CommitBlock")

				stateHash, err := state.CommitBlock(commit.Block)

				commit.Respond(stateHash, err)

			case snapshotRequest := <-babbleProxy.SnapshotRequestCh():
				t.Log("GetSnapshot")

				snapshot, err := state.GetSnapshot(snapshotRequest.BlockIndex)

				snapshotRequest.Respond(snapshot, err)

			case restoreRequest := <-babbleProxy.RestoreCh():
				t.Log("Restore")

				stateHash, err := state.Restore(restoreRequest.Snapshot)

				restoreRequest.Respond(stateHash, err)
			}
		}
	}()

	//create a few blocks
	blocks := [5]hashgraph.Block{}

	for i := 0; i < 5; i++ {
		blocks[i] = hashgraph.NewBlock(i, i+1, []byte{}, [][]byte{[]byte(fmt.Sprintf("block %d transaction", i))})
	}

	//commit first block and check that the client's statehash is correct
	stateHash, err := appProxy.CommitBlock(blocks[0])

	if err != nil {
		t.Fatal(err)
	}

	expectedStateHash := initialStateHash

	for _, t := range blocks[0].Transactions() {
		tHash := bcrypto.SHA256(t)

		expectedStateHash = bcrypto.SimpleHashFromTwoHashes(expectedStateHash, tHash)
	}

	if !reflect.DeepEqual(stateHash, expectedStateHash) {
		t.Fatalf("StateHash should be %v, not %v", expectedStateHash, stateHash)
	}

	snapshot, err := appProxy.GetSnapshot(blocks[0].Index())

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(snapshot, expectedStateHash) {
		t.Fatalf("Snapshot should be %v, not %v", expectedStateHash, snapshot)
	}

	//commit a few more blocks, then attempt to restore back to block 0 state
	for i := 1; i < 5; i++ {
		_, err := appProxy.CommitBlock(blocks[i])

		if err != nil {
			t.Fatal(err)
		}
	}

	err = appProxy.Restore(snapshot)

	if err != nil {
		t.Fatalf("Error restoring snapshot: %v", err)
	}

}
