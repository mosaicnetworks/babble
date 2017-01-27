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
	hg "github.com/arrivets/go-swirlds/hashgraph"
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

/*
|  e12  |
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
func initHashgraph(nodes []Node, keys []*ecdsa.PrivateKey, index map[string]string, participant int) {

	for i := 0; i < len(nodes); i++ {
		if i != participant {
			event := nodes[i].GetEvent(index[fmt.Sprintf("e%d", i)])
			if err := nodes[participant].InsertEvent(event); err != nil {
				fmt.Printf("error inserting %s: %s\n", getName(index, event.Hex()), err)
			}
		}
	}

	event01 := hg.NewEvent([][]byte{},
		[]string{index["e0"], index["e1"]}, //e0 and e1
		nodes[0].PubKey())
	if err := insertEvent(nodes, keys, index, event01, "e01", participant, 0); err != nil {
		fmt.Printf("error inserting e01: %s\n", err)
	}

	event20 := hg.NewEvent([][]byte{},
		[]string{index["e2"], index["e01"]}, //e2 and e01
		nodes[2].PubKey())
	if err := insertEvent(nodes, keys, index, event20, "e20", participant, 2); err != nil {
		fmt.Printf("error inserting e20: %s\n", err)
	}

	event12 := hg.NewEvent([][]byte{},
		[]string{index["e1"], index["e20"]}, //e1 and e20
		nodes[1].PubKey())
	if err := insertEvent(nodes, keys, index, event12, "e12", participant, 1); err != nil {
		fmt.Printf("error inserting e12: %s\n", err)
	}
}

func insertEvent(nodes []Node, keys []*ecdsa.PrivateKey, index map[string]string,
	event hg.Event, name string, particant int, creator int) error {

	if particant == creator {
		if err := nodes[particant].SignAndInsertSelfEvent(event); err != nil {
			return err
		}
		//event is not signed because passed by value
		index[name] = nodes[particant].Head
	} else {
		event.Sign(keys[creator])
		if err := nodes[particant].InsertEvent(event); err != nil {
			return err
		}
		index[name] = event.Hex()
	}
	return nil
}

func TestDiff(t *testing.T) {
	nodes, keys, index := initNodes()

	initHashgraph(nodes, keys, index, 0)

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

	knownBy1 := nodes[1].Known()
	head, unknownBy1 := nodes[0].Diff(knownBy1)
	if head != index["e01"] {
		t.Fatalf("head of node 0 should be e01")
	}

	if l := len(unknownBy1); l != 5 {
		t.Fatalf("length of unknown should be 5, not %d", l)
	}

	expectedOrder := []string{"e0", "e2", "e01", "e20", "e12"}
	for i, e := range unknownBy1 {
		if name := getName(index, e.Hex()); name != expectedOrder[i] {
			t.Fatalf("element %d should be %s, not %s", i, expectedOrder[i], name)
		}
	}

}

func TestSync(t *testing.T) {
	nodes, _, index := initNodes()

	/*
	   node 0           node 1          node 2

	   e0  |   |        |   e1  |       |   |   e2
	   0   1   2        0   1   2       0   1   2
	*/

	//node 1 is going to tell node 0 everything it knows
	if err := sync(nodes, 1, 0); err != nil {
		t.Fatal(err)
	}

	/*
	   node 0           node 1          node 2

	   e01 |   |
	   | \ |   |
	   e0  e1  |        |   e1  |       |   |   e2
	   0   1   2        0   1   2       0   1   2
	*/

	knownBy0 := nodes[0].Known()
	if k := knownBy0[fmt.Sprintf("0x%X", nodes[0].PubKey())]; k != 2 {
		t.Fatalf("node 0 should have 2 events for node 0, not %d", k)
	}
	if k := knownBy0[fmt.Sprintf("0x%X", nodes[1].PubKey())]; k != 1 {
		t.Fatalf("node 0 should have 1 events for node 1, not %d", k)
	}
	if k := knownBy0[fmt.Sprintf("0x%X", nodes[2].PubKey())]; k != 0 {
		t.Fatalf("node 0 should have 0 events for node 2, not %d", k)
	}
	node0Head := nodes[0].GetHead()
	if node0Head.Body.Parents[0] != index["e0"] {
		t.Fatalf("node 0 head self-parent should be e0")
	}
	if node0Head.Body.Parents[1] != index["e1"] {
		t.Fatalf("node 0 head other-parent should be e1")
	}
	index["e01"] = node0Head.Hex()

	//node 0 is going to tell node 2 everything it knows
	if err := sync(nodes, 0, 2); err != nil {
		t.Fatal(err)
	}

	/*

	   node 0           node 1          node 2

	                                    |   |  e20
	                                    |   | / |
	                                    |   /   |
	                                    | / |   |
	   e01 |   |                        e01 |   |
	   | \ |   |                        | \ |   |
	   e0  e1  |        |   e1  |       e0  e1  e2
	   0   1   2        0   1   2       0   1   2
	*/

	knownBy2 := nodes[2].Known()
	if k := knownBy2[fmt.Sprintf("0x%X", nodes[0].PubKey())]; k != 2 {
		t.Fatalf("node 2 should have 2 events for node 0, not %d", k)
	}
	if k := knownBy2[fmt.Sprintf("0x%X", nodes[1].PubKey())]; k != 1 {
		t.Fatalf("node 2 should have 1 events for node 1, not %d", k)
	}
	if k := knownBy2[fmt.Sprintf("0x%X", nodes[2].PubKey())]; k != 2 {
		t.Fatalf("node 2 should have 2 events for node 2, not %d", k)
	}
	node2Head := nodes[2].GetHead()
	if node2Head.Body.Parents[0] != index["e2"] {
		t.Fatalf("node 2 head self-parent should be e2")
	}
	if node2Head.Body.Parents[1] != index["e01"] {
		t.Fatalf("node 2 head other-parent should be e01")
	}
	index["e20"] = node2Head.Hex()

	//node 2 is going to tell node 1 everything it knows
	if err := sync(nodes, 2, 1); err != nil {
		t.Fatal(err)
	}

	/*

	   node 0           node 1          node 2

	                    |  e12  |
	                    |   | \ |
	                    |   |  e20      |   |  e20
	                    |   | / |       |   | / |
	                    |   /   |       |   /   |
	                    | / |   |       | / |   |
	   e01 |   |        e01 |   |       e01 |   |
	   | \ |   |        | \ |   |       | \ |   |
	   e0  e1  |        e0  e1  e2      e0  e1  e2
	   0   1   2        0   1   2       0   1   2
	*/

	knownBy1 := nodes[1].Known()
	if k := knownBy1[fmt.Sprintf("0x%X", nodes[0].PubKey())]; k != 2 {
		t.Fatalf("node 1 should have 2 events for node 0, not %d", k)
	}
	if k := knownBy1[fmt.Sprintf("0x%X", nodes[1].PubKey())]; k != 2 {
		t.Fatalf("node 1 should have 2 events for node 1, not %d", k)
	}
	if k := knownBy1[fmt.Sprintf("0x%X", nodes[2].PubKey())]; k != 2 {
		t.Fatalf("node 1 should have 2 events for node 2, not %d", k)
	}
	node1Head := nodes[1].GetHead()
	if node1Head.Body.Parents[0] != index["e1"] {
		t.Fatalf("node 1 head self-parent should be e1")
	}
	if node1Head.Body.Parents[1] != index["e20"] {
		t.Fatalf("node 1 head other-parent should be e20")
	}
	index["e12"] = node1Head.Hex()

}

func sync(nodes []Node, from int, to int) error {
	knownByTo := nodes[to].Known()
	toHead, unknownByTo := nodes[from].Diff(knownByTo)
	return nodes[to].Sync(toHead, unknownByTo)
}

func getName(index map[string]string, hash string) string {
	for name, h := range index {
		if h == hash {
			return name
		}
	}
	return ""
}
