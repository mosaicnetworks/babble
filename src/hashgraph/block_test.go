package hashgraph

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

func createTestBlock() *Block {
	block := NewBlock(0, 1,
		[]byte("framehash"),
		[]*peers.Peer{},
		[][]byte{
			[]byte("abc"),
			[]byte("def"),
			[]byte("ghi"),
		},
		[]InternalTransaction{
			NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1", "paris", "peer1")),
			NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2", "london", "peer2")),
		},
		0,
	)

	receipts := []InternalTransactionReceipt{}
	for _, itx := range block.InternalTransactions() {
		receipts = append(receipts, itx.AsAccepted())
	}
	block.Body.InternalTransactionReceipts = receipts

	return block
}

func TestSignBlock(t *testing.T) {
	privateKey, _ := keys.GenerateECDSAKey()

	block := createTestBlock()

	sig, err := block.Sign(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	res, err := block.Verify(sig)
	if err != nil {
		t.Fatalf("Error verifying signature: %v", err)
	}
	if !res {
		t.Fatal("Verify returned false")
	}
}

func TestAppendSignature(t *testing.T) {
	privateKey, _ := keys.GenerateECDSAKey()

	block := createTestBlock()

	sig, err := block.Sign(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	err = block.SetSignature(sig)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature, err := block.GetSignature(keys.PublicKeyHex(&privateKey.PublicKey))
	if err != nil {
		t.Fatal(err)
	}

	res, err := block.Verify(blockSignature)
	if err != nil {
		t.Fatalf("Error verifying signature: %v", err)
	}
	if !res {
		t.Fatal("Verify returned false")
	}
}

func TestNewBlockFromFrame(t *testing.T) {

	timestamps := []int64{
		rand.Int63(),
		rand.Int63(),
		rand.Int63(),
	}
	transactions := [][]byte{
		[]byte("transaction1"),
		[]byte("transaction2"),
		[]byte("transaction3"),
		[]byte("transaction4"),
		[]byte("transaction5"),
		[]byte("transaction6"),
		[]byte("transaction7"),
		[]byte("transaction8"),
		[]byte("transaction9"),
	}
	internalTransactions := []InternalTransaction{
		NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1000.pub", "peer1000.addr", "peer1000")),
		NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1001.pub", "peer1001.addr", "peer1001")),
		NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1002.pub", "peer1002.addr", "peer1002")),
	}

	frame := &Frame{
		Round: 56,
		Peers: []*peers.Peer{
			peers.NewPeer("peer1.pub", "peer1.addr", "peer1"),
			peers.NewPeer("peer2.pub", "peer2.addr", "peer2"),
			peers.NewPeer("peer3.pub", "peer3.addr", "peer3"),
		},
		Roots: nil,
		Events: []*FrameEvent{
			{
				Core: &Event{
					Body: EventBody{
						Transactions:         transactions[0:3],
						InternalTransactions: internalTransactions[:1],
						Timestamp:            timestamps[0],
					},
				},
			},
			{
				Core: &Event{
					Body: EventBody{
						Transactions:         transactions[3:6],
						InternalTransactions: internalTransactions[1:2],
						Timestamp:            timestamps[1],
					},
				},
			},
			{
				Core: &Event{
					Body: EventBody{
						Transactions:         transactions[6:],
						InternalTransactions: internalTransactions[2:],
						Timestamp:            timestamps[2],
					},
				},
			},
		},
	}

	block, err := NewBlockFromFrame(10, frame)
	if err != nil {
		t.Fatal(err)
	}

	if block.Index() != 10 {
		t.Fatalf("block index should be %d, not %d", 0, block.Index())
	}

	if block.RoundReceived() != frame.Round {
		t.Fatalf("block round received should be %d, not %d", frame.Round, block.RoundReceived())
	}

	frameHash, _ := frame.Hash()
	if !reflect.DeepEqual(block.FrameHash(), frameHash) {
		t.Fatalf("block frame hash should be %v, not %v", frameHash, block.FrameHash())
	}

	if !reflect.DeepEqual(block.Transactions(), transactions) {
		t.Fatal("block has wrong transactions")
	}

	if !reflect.DeepEqual(block.InternalTransactions(), internalTransactions) {
		t.Fatal("block has wrong internal transactions")
	}

	median := common.Median(timestamps)
	if block.Timestamp() != median {
		t.Fatalf("block timestamp should be %d, not %d", median, block.Timestamp())
	}
}
