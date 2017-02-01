/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package net

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"crypto/ecdsa"

	"reflect"

	scrypto "github.com/arrivets/go-swirlds/crypto"
)

func TestJSONPeers(t *testing.T) {
	// Create a test dir
	dir, err := ioutil.TempDir("", "swirld")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}
	defer os.RemoveAll(dir)

	// Create the store
	store := NewJSONPeers(dir)

	// Try a read, should get nothing
	peers, err := store.Peers()
	if err != nil {
		t.Fatalf("err: %v", err)
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

	// Try a read, should peers
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
