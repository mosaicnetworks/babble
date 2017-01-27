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
package node

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/arrivets/go-swirlds/crypto"
	"github.com/arrivets/go-swirlds/hashgraph"
)

func TestInit(t *testing.T) {
	key, _ := crypto.GenerateECDSAKey()
	participants := []string{
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
	}
	node := NewNode(key, participants)
	if err := node.Init(); err != nil {
		t.Fatalf("Init returned and error: %s", err)
	}
}

func initNodes() ([]Node, []*ecdsa.PrivateKey, map[string]string) {
	n := 3
	nodes := []Node{}
	index := make(map[string]string)

	participantKeys := []*ecdsa.PrivateKey{}
	participantPubs := []string{}
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		participantKeys = append(participantKeys, key)
		participantPubs = append(participantPubs,
			fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)))
	}

	for i := 0; i < n; i++ {
		node := NewNode(participantKeys[i], participantPubs)
		node.Init()
		nodes = append(nodes, node)
		index[fmt.Sprintf("e%d", i)] = node.Head
	}

	return nodes, participantKeys, index
}

func TestAskDiff(t *testing.T) {
	nodes, keys, index := initNodes()

	event1 := nodes[1].GetHead()
	if err := nodes[0].InsertEvent(event1); err != nil {
		t.Fatalf("error inserting e1: %s", err)
	}

	event2 := nodes[2].GetHead()
	if err := nodes[0].InsertEvent(event2); err != nil {
		t.Fatalf("error inserting e2: %s", err)
	}

	event01 := hashgraph.NewEvent([][]byte{},
		[]string{index["e0"], index["e1"]}, //e0 and e1
		nodes[0].PubKey())
	if err := nodes[0].SignAndInsertSelfEvent(event01); err != nil {
		t.Fatalf("error inserting e01: %s", err)
	}
	//event01 is not signed because passed by value
	index["e01"] = nodes[0].Head

	event20 := hashgraph.NewEvent([][]byte{},
		[]string{index["e2"], index["e01"]}, //e2 and e01
		nodes[2].PubKey())
	event20.Sign(keys[2])
	if err := nodes[0].InsertEvent(event20); err != nil {
		t.Fatalf("error inserting e20: %s", err)
	}
	index["e20"] = event20.Hex()

	event12 := hashgraph.NewEvent([][]byte{},
		[]string{index["e1"], index["e20"]}, //e1 and e20
		nodes[1].PubKey())
	event12.Sign(keys[1])
	if err := nodes[0].InsertEvent(event12); err != nil {
		t.Fatalf("error inserting e12: %s", err)
	}
	index["e12"] = event12.Hex()

	/*
	   P0 knows

	   |  e12  |
	   |   | \ |
	   |   |   e20
	   |   | / |
	   |   /   |
	   | / |   |
	   e01 |   |        P1 knows
	   | \ |   |
	   e0  e1  e2       |   e1  |
	   0   1   2        0   1   2
	*/

	known := nodes[1].Known()
	head, otherUnknown := nodes[0].AskDiff(known)

	if head.Hex() != index["e01"] {
		t.Fatalf("head of node 0 should be e01")
	}

	if l := len(otherUnknown); l != 5 {
		t.Fatalf("length of unknown should be 5, not %d", l)
	}

	expectedOrder := []string{"e0", "e2", "e01", "e20", "e12"}
	for i, e := range otherUnknown {
		if name := getName(index, e.Hex()); name != expectedOrder[i] {
			t.Fatalf("element %d should be %s, not %s", i, expectedOrder[i], name)
		}
	}

}

func getName(index map[string]string, hash string) string {
	for name, h := range index {
		if h == hash {
			return name
		}
	}
	return ""
}
