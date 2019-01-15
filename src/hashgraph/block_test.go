package hashgraph

import (
	"fmt"
	"testing"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestSignBlock(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()

	block := NewBlock(0, 1,
		[]byte("framehash"),
		[]*peers.Peer{},
		[][]byte{
			[]byte("abc"),
			[]byte("def"),
			[]byte("ghi"),
		},
		[]InternalTransaction{
			NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1", "paris")),
			NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2", "london")),
		})

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
	privateKey, _ := crypto.GenerateECDSAKey()
	pubKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	block := NewBlock(0, 1,
		[]byte("framehash"),
		[]*peers.Peer{},
		[][]byte{
			[]byte("abc"),
			[]byte("def"),
			[]byte("ghi"),
		},
		[]InternalTransaction{
			NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1", "paris")),
			NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2", "london")),
		})

	sig, err := block.Sign(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	err = block.SetSignature(sig)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature, err := block.GetSignature(fmt.Sprintf("0x%X", pubKeyBytes))
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
