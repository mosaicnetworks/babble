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

	"strings"

	"github.com/arrivets/go-swirlds/crypto"
)

type Node struct {
	Pub    []byte
	PubHex string
	Key    *ecdsa.PrivateKey
	Events []Event
}

func NewNode(key *ecdsa.PrivateKey) Node {
	pub := crypto.FromECDSAPub(&key.PublicKey)
	node := Node{
		Key:    key,
		Pub:    pub,
		PubHex: fmt.Sprintf("0x%X", pub),
		Events: []Event{},
	}
	return node
}
func (node *Node) signAndAddEvent(event Event, name string, index map[string]string, orderedEvents *[]Event) {
	event.Sign(node.Key)
	node.Events = append(node.Events, event)
	index[name] = event.Hex()
	*orderedEvents = append(*orderedEvents, event)
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
func initHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key)
		event := NewEvent([][]byte{}, []string{"", ""}, node.Pub)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

	event01 := NewEvent([][]byte{},
		[]string{index["e0"], index["e1"]}, //e0 and e1
		nodes[0].Pub)
	nodes[0].signAndAddEvent(event01, "e01", index, orderedEvents)

	event20 := NewEvent([][]byte{},
		[]string{index["e2"], index["e01"]}, //e2 and e01
		nodes[2].Pub)
	nodes[2].signAndAddEvent(event20, "e20", index, orderedEvents)

	event12 := NewEvent([][]byte{},
		[]string{index["e1"], index["e20"]}, //e1 and e20
		nodes[1].Pub)
	nodes[1].signAndAddEvent(event12, "e12", index, orderedEvents)

	participants := []string{}
	events := make(map[string]Event)
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
		for _, ev := range node.Events {
			events[ev.Hex()] = ev
		}
	}
	hashgraph := NewHashgraph(participants)
	hashgraph.Events = events
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

func TestSelfAncestor(t *testing.T) {
	h, index := initHashgraph()

	//1 generation
	if !h.SelfAncestor(index["e01"], index["e0"]) {
		t.Fatal("e0 should be self ancestor of e01")
	}
	if !h.SelfAncestor(index["e20"], index["e2"]) {
		t.Fatal("e2 should be self ancestor of e20")
	}
	if !h.SelfAncestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be self ancestor of e12")
	}

	//1 generation false negatives
	if h.SelfAncestor(index["e01"], index["e1"]) {
		t.Fatal("e1 should not be self ancestor of e01")
	}
	if h.SelfAncestor(index["e20"], index["e01"]) {
		t.Fatal("e01 should not be self ancestor of e20")
	}
	if h.SelfAncestor(index["e12"], index["e20"]) {
		t.Fatal("e20 should not be self ancestor of e12")
	}

	//2 generation false negative
	if h.SelfAncestor(index["e20"], index["e0"]) {
		t.Fatal("e0 should not be self ancestor of e20")
	}
	if h.SelfAncestor(index["e12"], index["e2"]) {
		t.Fatal("e2 should not be self ancestor of e12")
	}

}

/*
|   e12    |
|    | \   |
|    |   \ |
|    |    e20
|    |   / |
|    | /   |
|    /     |
|  / |     |
e01  |     |
| \  |     |
|   \|     |
|    |\    |
|    |  \  |
e0   e1 (a)e2
0    1     2

Node 2 Forks; events a and e2 are both created by node2, they are not self-parents
and yet they are both ancestors of event e20
*/
func initForkHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key)
		event := NewEvent([][]byte{}, []string{"", ""}, node.Pub)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

	//a and e2 need to have different hashes
	eventA := NewEvent([][]byte{[]byte("yo")}, []string{"", ""}, nodes[2].Pub)
	nodes[2].signAndAddEvent(eventA, "a", index, orderedEvents)

	event01 := NewEvent([][]byte{},
		[]string{index["e0"], index["a"]}, //e0 and a
		nodes[0].Pub)
	nodes[0].signAndAddEvent(event01, "e01", index, orderedEvents)

	event20 := NewEvent([][]byte{},
		[]string{index["e2"], index["e01"]}, //e2 and e01
		nodes[2].Pub)
	nodes[2].signAndAddEvent(event20, "e20", index, orderedEvents)

	event12 := NewEvent([][]byte{},
		[]string{index["e1"], index["e20"]}, //e1 and e20
		nodes[1].Pub)
	nodes[1].signAndAddEvent(event12, "e12", index, orderedEvents)

	participants := []string{}
	events := make(map[string]Event)
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
		for _, ev := range node.Events {
			events[ev.Hex()] = ev
		}
	}
	hashgraph := NewHashgraph(participants)
	hashgraph.Events = events
	return hashgraph, index
}

func TestDetectFork(t *testing.T) {
	h, index := initForkHashgraph()

	//1 generation
	fork := h.DetectFork(index["e20"], index["a"])
	if !fork {
		t.Fatal("e20 should detect a fork under a")
	}
	fork = h.DetectFork(index["e20"], index["e2"])
	if !fork {
		t.Fatal("e20 should detect a fork under e2")
	}
	fork = h.DetectFork(index["e12"], index["e20"])
	if !fork {
		t.Fatal("e12 should detect a fork under e20")
	}

	//2 generations
	fork = h.DetectFork(index["e12"], index["a"])
	if !fork {
		t.Fatal("e12 should detect a fork under a")
	}
	fork = h.DetectFork(index["e12"], index["e2"])
	if !fork {
		t.Fatal("e12 should detect a fork under e2")
	}

	//false negatives
	fork = h.DetectFork(index["e01"], index["e0"])
	if fork {
		t.Fatal("e01 should not detect a fork under e0")
	}
	fork = h.DetectFork(index["e01"], index["a"])
	if fork {
		t.Fatal("e01 should not detect a fork under 'a'")
	}
	fork = h.DetectFork(index["e01"], index["e2"])
	if fork {
		t.Fatal("e01 should not detect a fork under e2")
	}
	fork = h.DetectFork(index["e20"], index["e01"])
	if fork {
		t.Fatal("e20 should not detect a fork under e01")
	}
	fork = h.DetectFork(index["e12"], index["e01"])
	if fork {
		t.Fatal("e12 should not detect a fork under e01")
	}
}

func TestSee(t *testing.T) {
	h, index := initForkHashgraph()

	if !h.See(index["e01"], index["e0"]) {
		t.Fatal("e01 should see e0")
	}
	if !h.See(index["e01"], index["a"]) {
		t.Fatal("e01 should see 'a'")
	}
	if !h.See(index["e20"], index["e0"]) {
		t.Fatal("e20 should see e0")
	}
	if !h.See(index["e20"], index["e01"]) {
		t.Fatal("e20 should see e01")
	}
	if !h.See(index["e12"], index["e01"]) {
		t.Fatal("e12 should see e01")
	}
	if !h.See(index["e12"], index["e0"]) {
		t.Fatal("e12 should see e0")
	}
	if !h.See(index["e12"], index["e1"]) {
		t.Fatal("e12 should see e1")
	}

	//fork
	if h.See(index["e20"], index["a"]) {
		t.Fatal("e20 should not see 'a' because of fork")
	}
	if h.See(index["e20"], index["e2"]) {
		t.Fatal("e20 should not see e2 because of fork")
	}
	if h.See(index["e12"], index["a"]) {
		t.Fatal("e12 should not see 'a' because of fork")
	}
	if h.See(index["e12"], index["e2"]) {
		t.Fatal("e12 should not see e2 because of fork")
	}
	if h.See(index["e12"], index["e20"]) {
		t.Fatal("e12 should not see e20 because of fork")
	}

}

func TestStronglySee(t *testing.T) {
	h, index := initHashgraph()

	if !h.StronglySee(index["e12"], index["e0"]) {
		t.Fatalf("e12 should strongly see e0")
	}
	if !h.StronglySee(index["e12"], index["e1"]) {
		t.Fatalf("e12 should strongly see e1")
	}
	if !h.StronglySee(index["e12"], index["e01"]) {
		t.Fatalf("e12 should strongly see e01")
	}
	if !h.StronglySee(index["e20"], index["e1"]) {
		t.Fatalf("e20 should strongly see e1")
	}

	//false negatives
	if h.StronglySee(index["e12"], index["e2"]) {
		t.Fatalf("e12 should not strongly see e2")
	}
	if h.StronglySee(index["e12"], index["e20"]) {
		t.Fatalf("e12 should not strongly see e20")
	}
	if h.StronglySee(index["e20"], index["e01"]) {
		t.Fatalf("e20 should not strongly see e01")
	}
	if h.StronglySee(index["e20"], index["e0"]) {
		t.Fatalf("e20 should not strongly see e0")
	}
	if h.StronglySee(index["e20"], index["e2"]) {
		t.Fatalf("e20 should not strongly see e2")
	}
	if h.StronglySee(index["e01"], index["e0"]) {
		t.Fatalf("e01 should not strongly see e0")
	}
	if h.StronglySee(index["e01"], index["e1"]) {
		t.Fatalf("e01 should not strongly see e1")
	}
	if h.StronglySee(index["e01"], index["e2"]) {
		t.Fatalf("e01 should not strongly see e2")
	}
	if h.StronglySee(index["e0"], index["e0"]) {
		t.Fatalf("e0 should not strongly see e0")
	}

	//fork
	h, index = initForkHashgraph()
	if h.StronglySee(index["e12"], index["a"]) {
		t.Fatalf("e12 should not strongly see 'a' because of fork")
	}
	if h.StronglySee(index["e12"], index["e2"]) {
		t.Fatalf("e12 should not strongly see e2 because of fork")
	}

}

/*
|   f1  |
|  /|   |
e02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  e21
|   | / |
|  e10  |
| / |   |
e0  e1  e2
0   1    2
*/
func initRoundHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key)
		event := NewEvent([][]byte{}, []string{"", ""}, node.Pub)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

	event10 := NewEvent([][]byte{},
		[]string{index["e1"], index["e0"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(event10, "e10", index, orderedEvents)

	event21 := NewEvent([][]byte{},
		[]string{index["e2"], index["e10"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(event21, "e21", index, orderedEvents)

	event02 := NewEvent([][]byte{},
		[]string{index["e0"], index["e21"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(event02, "e02", index, orderedEvents)

	eventf1 := NewEvent([][]byte{},
		[]string{index["e10"], index["e02"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(eventf1, "f1", index, orderedEvents)

	participants := []string{}
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
	}

	hashgraph := NewHashgraph(participants)
	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}
	return hashgraph, index
}

func TestParentRound(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := make(map[string]Trilean)
	round0Witnesses[index["e0"]] = Undefined
	round0Witnesses[index["e1"]] = Undefined
	round0Witnesses[index["e2"]] = Undefined
	h.Rounds[0] = RoundInfo{Witnesses: round0Witnesses}

	round1Witnesses := make(map[string]Trilean)
	round1Witnesses[index["f1"]] = Undefined
	h.Rounds[1] = RoundInfo{Witnesses: round1Witnesses}

	if r := h.ParentRound(index["e0"]); r != 0 {
		t.Fatalf("parent round of e0 should be 0, not %d", r)
	}
	if r := h.ParentRound(index["e1"]); r != 0 {
		t.Fatalf("parent round of e1 should be 0, not %d", r)
	}
	if r := h.ParentRound(index["e10"]); r != 0 {
		t.Fatalf("parent round of e10 should be 0, not %d", r)
	}
	if r := h.ParentRound(index["f1"]); r != 0 {
		t.Fatalf("parent round of f1 should be 0, not %d", r)
	}
}

func TestWitness(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := make(map[string]Trilean)
	round0Witnesses[index["e0"]] = Undefined
	round0Witnesses[index["e1"]] = Undefined
	round0Witnesses[index["e2"]] = Undefined
	h.Rounds[0] = RoundInfo{Witnesses: round0Witnesses}

	round1Witnesses := make(map[string]Trilean)
	round1Witnesses[index["f1"]] = Undefined
	h.Rounds[1] = RoundInfo{Witnesses: round1Witnesses}

	if !h.Witness(index["e0"]) {
		t.Fatalf("e0 should be witness")
	}
	if !h.Witness(index["e1"]) {
		t.Fatalf("e1 should be witness")
	}
	if !h.Witness(index["e2"]) {
		t.Fatalf("e2 should be witness")
	}
	if !h.Witness(index["f1"]) {
		t.Fatalf("f1 should be witness")
	}

	if h.Witness(index["e10"]) {
		t.Fatalf("e10 should not be witness")
	}
	if h.Witness(index["e21"]) {
		t.Fatalf("e21 should not be witness")
	}
	if h.Witness(index["e02"]) {
		t.Fatalf("e02 should not be witness")
	}
}

func TestRoundInc(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := make(map[string]Trilean)
	round0Witnesses[index["e0"]] = Undefined
	round0Witnesses[index["e1"]] = Undefined
	round0Witnesses[index["e2"]] = Undefined
	h.Rounds[0] = RoundInfo{Witnesses: round0Witnesses}

	if !h.RoundInc(index["f1"]) {
		t.Fatal("RoundInc f1 should be true")
	}

	if h.RoundInc(index["e02"]) {
		t.Fatal("RoundInc e02 should be false because it doesnt strongly see e2")
	}
}

func TestRound(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := make(map[string]Trilean)
	round0Witnesses[index["e0"]] = Undefined
	round0Witnesses[index["e1"]] = Undefined
	round0Witnesses[index["e2"]] = Undefined
	h.Rounds[0] = RoundInfo{Witnesses: round0Witnesses}

	if r := h.Round(index["f1"]); r != 1 {
		t.Fatalf("round of f1 should be 1 not %d", r)
	}
	if r := h.Round(index["e02"]); r != 0 {
		t.Fatalf("round of e02 should be 0 not %d", r)
	}

}

func TestRoundDiff(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := make(map[string]Trilean)
	round0Witnesses[index["e0"]] = Undefined
	round0Witnesses[index["e1"]] = Undefined
	round0Witnesses[index["e2"]] = Undefined
	h.Rounds[0] = RoundInfo{Witnesses: round0Witnesses}

	if d, err := h.RoundDiff(index["f1"], index["e02"]); d != 1 {
		if err != nil {
			t.Fatalf("RoundDiff(f1, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(f1, e02) should be 1 not %d", d)
	}

	if d, err := h.RoundDiff(index["e02"], index["f1"]); d != -1 {
		if err != nil {
			t.Fatalf("RoundDiff(e02, f1) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e02, f1) should be -1 not %d", d)
	}
	if d, err := h.RoundDiff(index["e02"], index["e21"]); d != 0 {
		if err != nil {
			t.Fatalf("RoundDiff(e20, e21) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e20, e21) should be 0 not %d", d)
	}
}

func TestDivideRounds(t *testing.T) {
	h, index := initRoundHashgraph()

	h.DivideRounds()

	if l := len(h.Rounds); l != 2 {
		t.Fatalf("length of rounds should be 2 not %d", l)
	}

	round0 := h.Rounds[0]
	if l := len(round0.Witnesses); l != 3 {
		t.Fatalf("round 0 should have 3 witnesses, not %d", l)
	}
	if !contains(round0.Witnesses, index["e0"]) {
		t.Fatalf("round 0 witnesses should contain e0")
	}
	if !contains(round0.Witnesses, index["e1"]) {
		t.Fatalf("round 0 witnesses should contain e1")
	}
	if !contains(round0.Witnesses, index["e2"]) {
		t.Fatalf("round 0 witnesses should contain e2")
	}

	round1 := h.Rounds[1]
	if l := len(round1.Witnesses); l != 1 {
		t.Fatalf("round 1 should have 1 witness, not %d", l)
	}
	if !contains(round1.Witnesses, index["f1"]) {
		t.Fatalf("round 1 witnesses should contain f1")
	}

}

func contains(s map[string]Trilean, x string) bool {
	_, ok := s[x]
	return ok
}

/*
h0  |   h2
| \ | / |
|   h1  |
|  /|   |
g02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  g21
|   | / |
|  g10  |
| / |   |
g0  |   g2
| \ | / |
|   g1  |
|  /|   |
f02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  f21
|   | / |
|  f10  |
| / |   |
f0  |   f2
| \ | / |
|   f1  |
|  /|   |
e02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  e21
|   | / |
|  e10  |
| / |   |
e0  e1  e2
0   1    2
*/
func initConsensusHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key)
		event := NewEvent([][]byte{}, []string{"", ""}, node.Pub)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

	event10 := NewEvent([][]byte{},
		[]string{index["e1"], index["e0"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(event10, "e10", index, orderedEvents)

	event21 := NewEvent([][]byte{},
		[]string{index["e2"], index["e10"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(event21, "e21", index, orderedEvents)

	event02 := NewEvent([][]byte{},
		[]string{index["e0"], index["e21"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(event02, "e02", index, orderedEvents)

	eventf1 := NewEvent([][]byte{},
		[]string{index["e10"], index["e02"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(eventf1, "f1", index, orderedEvents)

	eventf0 := NewEvent([][]byte{},
		[]string{index["e02"], index["f1"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(eventf0, "f0", index, orderedEvents)

	eventf2 := NewEvent([][]byte{},
		[]string{index["e21"], index["f1"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(eventf2, "f2", index, orderedEvents)

	eventf10 := NewEvent([][]byte{},
		[]string{index["f1"], index["f0"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(eventf10, "f10", index, orderedEvents)

	eventf21 := NewEvent([][]byte{},
		[]string{index["f2"], index["f10"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(eventf21, "f21", index, orderedEvents)

	eventf02 := NewEvent([][]byte{},
		[]string{index["f0"], index["f21"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(eventf02, "f02", index, orderedEvents)

	eventg1 := NewEvent([][]byte{},
		[]string{index["f10"], index["f02"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(eventg1, "g1", index, orderedEvents)

	eventg0 := NewEvent([][]byte{},
		[]string{index["f02"], index["g1"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(eventg0, "g0", index, orderedEvents)

	eventg2 := NewEvent([][]byte{},
		[]string{index["f21"], index["g1"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(eventg2, "g2", index, orderedEvents)

	eventg10 := NewEvent([][]byte{},
		[]string{index["g1"], index["g0"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(eventg10, "g10", index, orderedEvents)

	eventg21 := NewEvent([][]byte{},
		[]string{index["g2"], index["g10"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(eventg21, "g21", index, orderedEvents)

	eventg02 := NewEvent([][]byte{},
		[]string{index["g0"], index["g21"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(eventg02, "g02", index, orderedEvents)

	eventh1 := NewEvent([][]byte{},
		[]string{index["g10"], index["g02"]},
		nodes[1].Pub)
	nodes[1].signAndAddEvent(eventh1, "h1", index, orderedEvents)

	eventh0 := NewEvent([][]byte{},
		[]string{index["g02"], index["h1"]},
		nodes[0].Pub)
	nodes[0].signAndAddEvent(eventh0, "h0", index, orderedEvents)

	eventh2 := NewEvent([][]byte{},
		[]string{index["g21"], index["h1"]},
		nodes[2].Pub)
	nodes[2].signAndAddEvent(eventh2, "h2", index, orderedEvents)

	participants := []string{}
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
	}

	hashgraph := NewHashgraph(participants)
	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}
	return hashgraph, index
}

func TestDecideFame(t *testing.T) {
	h, index := initConsensusHashgraph()

	h.DivideRounds()
	h.DecideFame()

	if r := h.Round(index["g0"]); r != 2 {
		t.Fatalf("g0 round should be 2, not %d", r)
	}
	if r := h.Round(index["g1"]); r != 2 {
		t.Fatalf("g1 round should be 2, not %d", r)
	}
	if r := h.Round(index["g2"]); r != 2 {
		t.Fatalf("g2 round should be 2, not %d", r)
	}

	if f := h.Rounds[0].Witnesses[index["e0"]]; f != True {
		t.Fatalf("e0 should be famous; got %s", f)
	}
	if f := h.Rounds[0].Witnesses[index["e1"]]; f != True {
		t.Fatalf("e1 should be famous; got %s", f)
	}
	if f := h.Rounds[0].Witnesses[index["e2"]]; f != True {
		t.Fatalf("e2 should be famous; got %s", f)
	}
}

func TestOldestSelfAncestorToSee(t *testing.T) {
	h, index := initConsensusHashgraph()

	if a := h.OldestSelfAncestorToSee(index["f0"], index["e1"]); a != index["e02"] {
		t.Fatalf("oldest self ancestor of f0 to see e1 should be e02 not %s", getName(index, a))
	}
	if a := h.OldestSelfAncestorToSee(index["f1"], index["e0"]); a != index["e10"] {
		t.Fatalf("oldest self ancestor of f1 to see e0 should be e10 not %s", getName(index, a))
	}
	if a := h.OldestSelfAncestorToSee(index["e21"], index["e1"]); a != index["e21"] {
		t.Fatalf("oldest self ancestor of e20 to see e1 should be e21 not %s", a)
	}
	if a := h.OldestSelfAncestorToSee(index["e2"], index["e1"]); a != "" {
		t.Fatalf("oldest self ancestor of e2 to see e1 should be '' not %s", a)
	}

}

func TestDecideRoundReceived(t *testing.T) {
	h, index := initConsensusHashgraph()

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()

	for name, hash := range index {
		e, _ := h.Events[hash]
		if rune(name[0]) == rune('e') {
			if r := e.roundReceived; r != 1 {
				t.Fatalf("%s round received should be 1 not %d", name, r)
			}
		}
	}

}

func TestFindOrder(t *testing.T) {
	h, index := initConsensusHashgraph()

	h.DivideRounds()
	h.DecideFame()
	h.FindOrder()

	if l := len(h.Consensus); l != 6 {
		t.Fatalf("length of consensus should be 6 not %d", l)
	}

	//events which have the same consensus timestamp are ordered by whitened signature
	//which is not deterministic.
	expected1 := []string{"e0", "e10", "e1", "e21", "e2", "e02"}
	expected2 := []string{"e0", "e1", "e10", "e2", "e21", "e02"}
	for i, e := range h.Consensus {
		if name := getName(index, e); name != expected1[i] && name != expected2[i] {
			more := ""
			if expected1[i] != expected2[i] {
				more = fmt.Sprintf("(or %s)", expected2[i])
			}
			t.Fatalf("consensus[%d] should be %s %s, not %s", i, expected1[i], more, name)
		}
	}
}

func TestHeight(t *testing.T) {
	h, _ := initConsensusHashgraph()

	if l := len(h.ParticipantEvents[h.Participants[0]]); l != 7 {
		t.Fatalf("0 should have 7 events, not %d", l)
	}
	if l := len(h.ParticipantEvents[h.Participants[1]]); l != 7 {
		t.Fatalf("1 should have 7 events, not %d", l)
	}
	if l := len(h.ParticipantEvents[h.Participants[2]]); l != 7 {
		t.Fatalf("2 should have 7 events, not %d", l)
	}
}

func TestKnown(t *testing.T) {
	h, _ := initConsensusHashgraph()
	known := h.Known()
	if l := known[h.Participants[0]]; l != 7 {
		t.Fatalf("0 should have 7 events, not %d", l)
	}
	if l := known[h.Participants[1]]; l != 7 {
		t.Fatalf("1 should have 7 events, not %d", l)
	}
	if l := known[h.Participants[2]]; l != 7 {
		t.Fatalf("2 should have 7 events, not %d", l)
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

func disp(index map[string]string, events []string) string {
	names := []string{}
	for _, h := range events {
		names = append(names, getName(index, h))
	}
	return fmt.Sprintf("[%s]", strings.Join(names, " "))
}
