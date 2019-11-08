package inmem

import (
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

type TestProxy struct {
	*InmemProxy
	transactions [][]byte
	logger       *logrus.Entry
}

func (p *TestProxy) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	p.logger.Debug("CommitBlock")

	p.transactions = append(p.transactions, block.Transactions()...)

	receipts := []hashgraph.InternalTransactionReceipt{}
	for _, it := range block.InternalTransactions() {
		receipts = append(receipts, it.AsAccepted())
	}

	response := proxy.CommitResponse{
		StateHash:                   []byte("statehash"),
		InternalTransactionReceipts: receipts,
	}

	return response, nil
}

func (p *TestProxy) SnapshotHandler(blockIndex int) ([]byte, error) {
	p.logger.Debug("GetSnapshot")

	return []byte("snapshot"), nil
}

func (p *TestProxy) RestoreHandler(snapshot []byte) ([]byte, error) {
	p.logger.Debug("RestoreSnapshot")

	return []byte("statehash"), nil
}

func NewTestProxy(t *testing.T) *TestProxy {
	logger := common.NewTestEntry(t, common.TestLogLevel)

	proxy := &TestProxy{
		transactions: [][]byte{},
		logger:       logger,
	}

	proxy.InmemProxy = NewInmemProxy(proxy, logger)

	return proxy
}

func TestInmemProxyAppSide(t *testing.T) {
	proxy := NewTestProxy(t)

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
	proxy := NewTestProxy(t)

	transactions := [][]byte{
		[]byte("tx 1"),
		[]byte("tx 2"),
		[]byte("tx 3"),
	}

	block := hashgraph.NewBlock(0, 1, []byte{}, []*peers.Peer{}, transactions, []hashgraph.InternalTransaction{})

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

	if !reflect.DeepEqual(transactions, proxy.transactions) {
		t.Fatalf("Transactions should be %v, not %v", transactions, proxy.transactions)
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
}
