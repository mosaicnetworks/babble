package hashgraph

import (
	"testing"

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
		})

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
