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
	"io/ioutil"
	"os"
	"testing"
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

	// Initialize some peers
	newPeers := []Peer{
		Peer{NetAddr: "addr0", PubKey: "pubkey0"},
		Peer{NetAddr: "addr1", PubKey: "pubkey1"},
		Peer{NetAddr: "addr2", PubKey: "pubkey2"},
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
	if rp0A := peers[0].NetAddr; rp0A != newPeers[0].NetAddr {
		t.Fatalf("peers[0] NetAddr should be %s, not %s", newPeers[0].NetAddr, rp0A)
	}
	if rp0P := peers[0].PubKey; rp0P != newPeers[0].PubKey {
		t.Fatalf("peers[0] PubKey should be %s, not %s", newPeers[0].PubKey, rp0P)
	}
}
