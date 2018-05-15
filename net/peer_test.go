package net

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"crypto/ecdsa"

	"reflect"

	scrypto "github.com/mosaicnetworks/babble/crypto"
)

func TestJSONPeers(t *testing.T) {
	// Create a test dir
	dir, err := ioutil.TempDir("", "babble")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}
	defer os.RemoveAll(dir)

	// Create the store
	store := NewJSONPeers(dir)

	// Try a read, should get nothing
	peers, err := store.Peers()
	if err == nil {
		t.Fatalf("store.Peers() should generate an error")
	}
	if len(peers) != 0 {
		t.Fatalf("peers: %v", peers)
	}

	keys := []*ecdsa.PrivateKey{}
	newPeers := []Peer{}
	for i := 0; i < 3; i++ {
		key, _ := scrypto.GenerateECDSAKey()
		peer := Peer{
			NetAddr:   fmt.Sprintf("addr%d", i),
			PubKeyHex: fmt.Sprintf("0x%X", scrypto.FromECDSAPub(&key.PublicKey)),
		}
		keys = append(keys, key)
		newPeers = append(newPeers, peer)
	}

	if err := store.SetPeers(newPeers); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Try a read, should find 3 peers
	peers, err = store.Peers()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(peers) != 3 {
		t.Fatalf("peers: %v", peers)
	}

	for i := 0; i < 3; i++ {
		if peers[i].NetAddr != newPeers[i].NetAddr {
			t.Fatalf("peers[%d] NetAddr should be %s, not %s", i,
				newPeers[i].NetAddr, peers[i].NetAddr)
		}
		if peers[i].PubKeyHex != newPeers[i].PubKeyHex {
			t.Fatalf("peers[%d] PubKeyHex should be %s, not %s", i,
				newPeers[i].PubKeyHex, peers[i].PubKeyHex)
		}
		pubKeyBytes, err := peers[i].PubKeyBytes()
		if err != nil {
			t.Fatal(err)
		}
		pubKey := scrypto.ToECDSAPub(pubKeyBytes)
		if !reflect.DeepEqual(*pubKey, keys[i].PublicKey) {
			t.Fatalf("peers[%d] PublicKey not parsed correctly", i)
		}
	}
}
