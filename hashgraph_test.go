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
package hashgraph

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/arrivets/go-swirlds/crypto"
)

/*
|   e12 |
|   | \ |
|   |   e20
|   | / |
|   /   |
| / |   |
e01 |   |
| \ |   |
e0  e1  e2
0   1   2
*/
func initHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)

	nodes := []struct {
		Pub    []byte
		PubHex string
		Key    *ecdsa.PrivateKey
		Events []Event
	}{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pub := crypto.FromECDSAPub(&key.PublicKey)
		pubHex := fmt.Sprintf("0x%X", pub)
		event := NewEvent([][]byte{}, []string{"", ""}, pub)
		event.Sign(key)
		name := fmt.Sprintf("e%d", i)
		index[name] = event.Hex()
		events := []Event{event}
		node := struct {
			Pub    []byte
			PubHex string
			Key    *ecdsa.PrivateKey
			Events []Event
		}{Pub: pub, PubHex: pubHex, Key: key, Events: events}
		nodes = append(nodes, node)
	}

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[1].Events[0].Hex()},
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()

	event20 := NewEvent([][]byte{},
		[]string{nodes[0].Events[1].Hex(), nodes[2].Events[0].Hex()},
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[1].Hex()},
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()

	hashgraph := NewHashgraph()
	for _, node := range nodes {
		for _, ev := range node.Events {
			hashgraph.Events[ev.Hex()] = ev
		}
	}
	return hashgraph, index
}

func TestAncestor(t *testing.T) {
	h, index := initHashgraph()

	//1 generation
	if !h.Ancestor(index["e01"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e01")
	}
	if !h.Ancestor(index["e01"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e01")
	}
	if !h.Ancestor(index["e20"], index["e01"]) {
		t.Fatal("e01 should be ancestor of e20")
	}
	if !h.Ancestor(index["e20"], index["e2"]) {
		t.Fatal("e2 should be ancestor of e20")
	}
	if !h.Ancestor(index["e12"], index["e20"]) {
		t.Fatal("e20 should be ancestor of e12")
	}
	if !h.Ancestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e12")
	}

	//2 generations
	if !h.Ancestor(index["e20"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e20")
	}
	if !h.Ancestor(index["e20"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e20")
	}
	if !h.Ancestor(index["e12"], index["e01"]) {
		t.Fatal("e01 should be ancestor of e12")
	}
	if !h.Ancestor(index["e12"], index["e2"]) {
		t.Fatal("e2 should be ancestor of e12")
	}

	//3 generations
	if !h.Ancestor(index["e12"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e12")
	}
	if !h.Ancestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e12")
	}

	//false positive
	if h.Ancestor(index["e01"], index["e2"]) {
		t.Fatal("e2 should not be ancestor of e01")
	}

}
