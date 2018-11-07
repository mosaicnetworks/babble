package socket

import (
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/socket/app"
	bproxy "github.com/mosaicnetworks/babble/src/proxy/socket/babble"
	"github.com/sirupsen/logrus"
)

type TestHandler struct {
	blocks     []hashgraph.Block
	blockIndex int
	snapshot   []byte
	logger     *logrus.Logger
}

func (p *TestHandler) CommitHandler(block hashgraph.Block) ([]byte, error) {
	p.logger.Debug("CommitBlock")

	p.blocks = append(p.blocks, block)

	return []byte("statehash"), nil
}

func (p *TestHandler) SnapshotHandler(blockIndex int) ([]byte, error) {
	p.logger.Debug("GetSnapshot")

	p.blockIndex = blockIndex

	return []byte("snapshot"), nil
}

func (p *TestHandler) RestoreHandler(snapshot []byte) ([]byte, error) {
	p.logger.Debug("RestoreSnapshot")

	p.snapshot = snapshot

	return []byte("statehash"), nil
}

func NewTestHandler(t *testing.T) *TestHandler {
	logger := common.NewTestLogger(t)

	return &TestHandler{
		blocks:     []hashgraph.Block{},
		blockIndex: 0,
		snapshot:   []byte{},
		logger:     logger,
	}
}
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
	babbleProxy, err := bproxy.NewSocketBabbleProxy(proxyAddr, clientAddr, NewTestHandler(t), 1*time.Second, common.NewTestLogger(t))

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

	handler := NewTestHandler(t)

	//create babble proxy
	_, err = bproxy.NewSocketBabbleProxy(proxyAddr, clientAddr, handler, 1*time.Second, logger)

	transactions := [][]byte{
		[]byte("tx 1"),
		[]byte("tx 2"),
		[]byte("tx 3"),
	}

	block := hashgraph.NewBlock(0, 1, []byte{}, []*peers.Peer{}, transactions)
	expectedStateHash := []byte("statehash")
	expectedSnapshot := []byte("snapshot")

	stateHash, err := appProxy.CommitBlock(*block)
	if err != nil {
		t.Fatal(err)
	}

	handler.blocks[0].PeerSet = nil
	block.PeerSet = nil

	if !reflect.DeepEqual(*block, handler.blocks[0]) {
		t.Fatalf("block should be \n%#v\n, not \n%#v\n", *block, handler.blocks[0])
	}

	if !reflect.DeepEqual(stateHash, expectedStateHash) {
		t.Fatalf("StateHash should be %v, not %v", expectedStateHash, stateHash)
	}

	snapshot, err := appProxy.GetSnapshot(block.Index())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(block.Index(), handler.blockIndex) {
		t.Fatalf("blockIndex should be %v, not %v", block.Index(), handler.blockIndex)
	}

	if !reflect.DeepEqual(snapshot, expectedSnapshot) {
		t.Fatalf("Snapshot should be %v, not %v", expectedSnapshot, snapshot)
	}

	err = appProxy.Restore(snapshot)
	if err != nil {
		t.Fatalf("Error restoring snapshot: %v", err)
	}

	if !reflect.DeepEqual(expectedSnapshot, handler.snapshot) {
		t.Fatalf("snapshot should be %v, not %v", expectedSnapshot, handler.snapshot)
	}
}
