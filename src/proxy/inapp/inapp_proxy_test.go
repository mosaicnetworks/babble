package inapp

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	bcrypto "github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy/proto"
)

func TestInappProxyAppSide(t *testing.T) {
	proxy := NewInappProxy(1*time.Second, common.NewTestLogger(t))

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

	err := proxy.SubmitTx(tx)

	if err != nil {
		t.Fatal(err)
	}
}

func TestInappProxyBabbleSide(t *testing.T) {
	proxy := NewInappProxy(1*time.Second, common.NewTestLogger(t))

	state := proto.NewState(proxy.logger)

	initialStateHash := []byte{}

	go func() {
		for {
			select {
			case commit := <-proxy.CommitCh():
				t.Log("CommitBlock")

				stateHash, err := state.CommitBlock(commit.Block)

				commit.Respond(stateHash, err)

			case snapshotRequest := <-proxy.SnapshotRequestCh():
				t.Log("GetSnapshot")

				snapshot, err := state.GetSnapshot(snapshotRequest.BlockIndex)

				snapshotRequest.Respond(snapshot, err)

			case restoreRequest := <-proxy.RestoreCh():
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
	stateHash, err := proxy.CommitBlock(blocks[0])

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

	snapshot, err := proxy.GetSnapshot(blocks[0].Index())

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(snapshot, expectedStateHash) {
		t.Fatalf("Snapshot should be %v, not %v", expectedStateHash, snapshot)
	}

	//commit a few more blocks, then attempt to restore back to block 0 state
	for i := 1; i < 5; i++ {
		_, err := proxy.CommitBlock(blocks[i])

		if err != nil {
			t.Fatal(err)
		}
	}

	err = proxy.Restore(snapshot)

	if err != nil {
		t.Fatalf("Error restoring snapshot: %v", err)
	}
}
