package socket

import (
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
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

	transactions := [][]byte{
		[]byte("tx 1"),
		[]byte("tx 2"),
		[]byte("tx 3"),
	}

	block := hashgraph.NewBlock(0, 1, []byte{}, transactions)
	expectedStateHash := []byte("statehash")
	expectedSnapshot := []byte("snapshot")

	go func() {
		for {
			select {
			case commit := <-babbleProxy.CommitCh():
				t.Log("CommitBlock")

				if !reflect.DeepEqual(block, commit.Block) {
					t.Fatalf("block should be %v, not %v", block, commit.Block)
				}

				commit.Respond(expectedStateHash, nil)

			case snapshotRequest := <-babbleProxy.SnapshotRequestCh():
				t.Log("GetSnapshot")

				if !reflect.DeepEqual(block.Index(), snapshotRequest.BlockIndex) {
					t.Fatalf("blockIndex should be %v, not %v", block.Index(), snapshotRequest.BlockIndex)
				}

				snapshotRequest.Respond(expectedSnapshot, err)

			case restoreRequest := <-babbleProxy.RestoreCh():
				t.Log("Restore")

				if !reflect.DeepEqual(expectedSnapshot, restoreRequest.Snapshot) {
					t.Fatalf("snapshot should be %v, not %v", expectedSnapshot, restoreRequest.Snapshot)
				}

				restoreRequest.Respond(expectedStateHash, err)
			}
		}
	}()

	stateHash, err := appProxy.CommitBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(stateHash, expectedStateHash) {
		t.Fatalf("StateHash should be %v, not %v", expectedStateHash, stateHash)
	}

	snapshot, err := appProxy.GetSnapshot(block.Index())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(snapshot, expectedSnapshot) {
		t.Fatalf("Snapshot should be %v, not %v", expectedSnapshot, snapshot)
	}

	err = appProxy.Restore(snapshot)
	if err != nil {
		t.Fatalf("Error restoring snapshot: %v", err)
	}

}
