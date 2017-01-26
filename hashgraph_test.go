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
		[]string{nodes[0].Events[0].Hex(), nodes[1].Events[0].Hex()}, //e0 and e1
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()

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

	//a and e2 need to have different hashes
	eventA := NewEvent([][]byte{[]byte("yo")}, []string{"", ""}, nodes[2].Pub)
	eventA.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventA)
	index["a"] = eventA.Hex()

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e0 and A
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[2].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()

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

/*    f1  |
|    /|   |
|   / |   |
|  / e12  |
| /   | \ |
e02   |   e20
|\    | / |
| \   |/  |
|  \  /   |
|   \/|   |
|   /\|   |
e01   \   |
| \   |\  |
|  \  | \ |
e0   e1   e2
0     1   2
*/
func initRoundHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)

	nodes := []struct {
		Pub    []byte
		PubHex string
		Key    *ecdsa.PrivateKey
		Events []Event
	}{}
	orderedEvents := []Event{}

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
		orderedEvents = append(orderedEvents, event)
	}

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[1].Events[0].Hex()}, //e0 and e1
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()
	orderedEvents = append(orderedEvents, event01)

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()
	orderedEvents = append(orderedEvents, event20)

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()
	orderedEvents = append(orderedEvents, event12)

	event02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[1].Hex(), nodes[2].Events[0].Hex()}, //e01 and e2
		nodes[0].Pub)
	event02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event02)
	index["e02"] = event02.Hex()
	orderedEvents = append(orderedEvents, event02)

	eventf1 := NewEvent([][]byte{},
		[]string{nodes[1].Events[1].Hex(), nodes[0].Events[2].Hex()}, //e12 and e02
		nodes[1].Pub)
	eventf1.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventf1)
	index["f1"] = eventf1.Hex()
	orderedEvents = append(orderedEvents, eventf1)

	participants := []string{}
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
	}

	hashgraph := NewHashgraph(participants)
	for i, ev := range orderedEvents {
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
	if r := h.ParentRound(index["e01"]); r != 0 {
		t.Fatalf("parent round of e01 should be 0, not %d", r)
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

	if h.Witness(index["e01"]) {
		t.Fatalf("e01 should not be witness")
	}
	if h.Witness(index["e02"]) {
		t.Fatalf("e02 should not be witness")
	}
	if h.Witness(index["e20"]) {
		t.Fatalf("e20 should not be witness")
	}
	if h.Witness(index["e12"]) {
		t.Fatalf("e12 should not be witness")
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

	if h.RoundInc(index["e12"]) {
		t.Fatal("RoundInc e12 should be false because it doesnt strongly see e2")
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
	if r := h.Round(index["e12"]); r != 0 {
		t.Fatalf("round of e12 should be 0 not %d", r)
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
	if d, err := h.RoundDiff(index["f1"], index["e02"]); d != 1 {
		if err != nil {
			t.Fatalf("RoundDiff(f1, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(f1, e02) should be 1 not %d", d)
	}
	if d, err := h.RoundDiff(index["e20"], index["e02"]); d != 0 {
		if err != nil {
			t.Fatalf("RoundDiff(e20, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e20, e02) should be 1 not %d", d)
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
|     |  g3
g0    | / |
| \   g1  |
|  \ /|   |
|   \ |   |
|  / f12  |
| /   | \ |
f02   |   f20
| \   | / |
|  \  |/  |
|   \ /   |
|    /    |
|   / \   |
|  /  |\  |
f01   | \ |
| \   |  \|
|  \  |   f2
|   \ | / |
f0    f1  |
| \  /|   |
|  \/ |   |
|  / e12  |
| /   | \ |
e02   |   e20
|\    | / |
| \   |/  |
|  \  /   |
|   \/|   |
|   /\|   |
e01   \   |
| \   |\  |
|  \  | \ |
e0   e1   e2
0     1   2
*/
func init2RoundHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)

	nodes := []struct {
		Pub    []byte
		PubHex string
		Key    *ecdsa.PrivateKey
		Events []Event
	}{}
	orderedEvents := []Event{}

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
		orderedEvents = append(orderedEvents, event)
	}

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[1].Events[0].Hex()}, //e0 and e1
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()
	orderedEvents = append(orderedEvents, event01)

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()
	orderedEvents = append(orderedEvents, event20)

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()
	orderedEvents = append(orderedEvents, event12)

	event02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[1].Hex(), nodes[2].Events[0].Hex()}, //e01 and e2
		nodes[0].Pub)
	event02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event02)
	index["e02"] = event02.Hex()
	orderedEvents = append(orderedEvents, event02)

	eventf1 := NewEvent([][]byte{},
		[]string{nodes[1].Events[1].Hex(), nodes[0].Events[2].Hex()}, //e12 and e02
		nodes[1].Pub)
	eventf1.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventf1)
	index["f1"] = eventf1.Hex()
	orderedEvents = append(orderedEvents, eventf1)

	eventf0 := NewEvent([][]byte{},
		[]string{nodes[0].Events[2].Hex(), nodes[1].Events[1].Hex()}, //e02 and e12
		nodes[0].Pub)
	eventf0.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventf0)
	index["f0"] = eventf0.Hex()
	orderedEvents = append(orderedEvents, eventf0)

	eventf2 := NewEvent([][]byte{},
		[]string{nodes[2].Events[1].Hex(), nodes[1].Events[2].Hex()}, //e20 and f1
		nodes[2].Pub)
	eventf2.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventf2)
	index["f2"] = eventf2.Hex()
	orderedEvents = append(orderedEvents, eventf2)

	eventf01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[3].Hex(), nodes[1].Events[2].Hex()}, //f0 and f1
		nodes[0].Pub)
	eventf01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventf01)
	index["f01"] = eventf01.Hex()
	orderedEvents = append(orderedEvents, eventf01)

	eventf20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[2].Hex(), nodes[0].Events[3].Hex()}, //f2 and f0
		nodes[2].Pub)
	eventf20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventf20)
	index["f20"] = eventf20.Hex()
	orderedEvents = append(orderedEvents, eventf20)

	eventf02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[4].Hex(), nodes[2].Events[2].Hex()}, //f01 and f2
		nodes[0].Pub)
	eventf02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventf02)
	index["f02"] = eventf02.Hex()
	orderedEvents = append(orderedEvents, eventf02)

	eventf12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[2].Hex(), nodes[2].Events[3].Hex()}, //f1 and f20
		nodes[1].Pub)
	eventf12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventf12)
	index["f12"] = eventf12.Hex()
	orderedEvents = append(orderedEvents, eventf12)

	eventg0 := NewEvent([][]byte{},
		[]string{nodes[0].Events[5].Hex(), nodes[1].Events[3].Hex()}, //f02 and f12
		nodes[0].Pub)
	eventg0.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventg0)
	index["g0"] = eventg0.Hex()
	orderedEvents = append(orderedEvents, eventg0)

	eventg1 := NewEvent([][]byte{},
		[]string{nodes[1].Events[3].Hex(), nodes[0].Events[5].Hex()}, //f12 and f02
		nodes[1].Pub)
	eventg1.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventg1)
	index["g1"] = eventg1.Hex()
	orderedEvents = append(orderedEvents, eventg1)

	eventg2 := NewEvent([][]byte{},
		[]string{nodes[2].Events[3].Hex(), nodes[1].Events[4].Hex()}, //f20 and g1
		nodes[2].Pub)
	eventg2.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventg2)
	index["g2"] = eventg2.Hex()
	orderedEvents = append(orderedEvents, eventg2)

	//add other nodes

	participants := []string{}
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
	}

	hashgraph := NewHashgraph(participants)
	for i, ev := range orderedEvents {
		if err := hashgraph.InsertEvent(ev); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}
	return hashgraph, index
}

func TestDecideFame(t *testing.T) {
	h, index := init2RoundHashgraph()

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
	h, index := init2RoundHashgraph()

	if a := h.OldestSelfAncestorToSee(index["f0"], index["e1"]); a != index["e01"] {
		t.Fatalf("oldest self ancestor of f0 to see e1 should be e01 not %s", getName(index, a))
	}
	if a := h.OldestSelfAncestorToSee(index["f1"], index["e0"]); a != index["e12"] {
		t.Fatalf("oldest self ancestor of f1 to see e0 should be e12 not %s", getName(index, a))
	}
	if a := h.OldestSelfAncestorToSee(index["e20"], index["e1"]); a != index["e20"] {
		t.Fatalf("oldest self ancestor of e20 to see e1 should be e20 not %s", a)
	}
	if a := h.OldestSelfAncestorToSee(index["e2"], index["e1"]); a != "" {
		t.Fatalf("oldest self ancestor of e2 to see e1 should be '' not %s", a)
	}

}

/*
|     |   h2
h0    | / |
| \   h1  |
|  \ /|   |
|   \ |   |
|  / g12  |
| /   | \ |
g02   |   g20
| \   | / |
|  \  |/  |
|   \ /   |
|    /|   |
|   / \   |
|  /  |\  |
g01   | \ |
| \   |  \|
|  \  |   g2
g0  \ | / |
| \   g1  |
|  \ /|   |
|   \ |   |
|  / f12  |
| /   | \ |
f02   |   f20
| \   | / |
|  \  |/  |
|   \ /   |
|    /|   |
|   / \   |
|  /  |\  |
f01   | \ |
| \   |  \|
|  \  |   f2
|   \ | / |
f0    f1  |
| \  /|   |
|  \/ |   |
|  / e12  |
| /   | \ |
e02   |   e20
|\    | / |
| \   |/  |
|  \  /   |
|   \/|   |
|   /\|   |
e01   \   |
| \   |\  |
|  \  | \ |
e0   e1   e2
0     1   2
*/
func initConsensusHashgraph() (Hashgraph, map[string]string) {
	n := 3
	index := make(map[string]string)

	nodes := []struct {
		Pub    []byte
		PubHex string
		Key    *ecdsa.PrivateKey
		Events []Event
	}{}
	orderedEvents := []Event{}

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
		orderedEvents = append(orderedEvents, event)
	}

	event01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[0].Hex(), nodes[1].Events[0].Hex()}, //e0 and e1
		nodes[0].Pub)
	event01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event01)
	index["e01"] = event01.Hex()
	orderedEvents = append(orderedEvents, event01)

	event20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[0].Hex(), nodes[0].Events[1].Hex()}, //e2 and e01
		nodes[2].Pub)
	event20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, event20)
	index["e20"] = event20.Hex()
	orderedEvents = append(orderedEvents, event20)

	event12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[0].Hex(), nodes[2].Events[1].Hex()}, //e1 and e20
		nodes[1].Pub)
	event12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event12)
	index["e12"] = event12.Hex()
	orderedEvents = append(orderedEvents, event12)

	event02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[1].Hex(), nodes[2].Events[0].Hex()}, //e01 and e2
		nodes[0].Pub)
	event02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event02)
	index["e02"] = event02.Hex()
	orderedEvents = append(orderedEvents, event02)

	eventf1 := NewEvent([][]byte{},
		[]string{nodes[1].Events[1].Hex(), nodes[0].Events[2].Hex()}, //e12 and e02
		nodes[1].Pub)
	eventf1.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventf1)
	index["f1"] = eventf1.Hex()
	orderedEvents = append(orderedEvents, eventf1)

	eventf0 := NewEvent([][]byte{},
		[]string{nodes[0].Events[2].Hex(), nodes[1].Events[1].Hex()}, //e02 and e12
		nodes[0].Pub)
	eventf0.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventf0)
	index["f0"] = eventf0.Hex()
	orderedEvents = append(orderedEvents, eventf0)

	eventf2 := NewEvent([][]byte{},
		[]string{nodes[2].Events[1].Hex(), nodes[1].Events[2].Hex()}, //e20 and f1
		nodes[2].Pub)
	eventf2.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventf2)
	index["f2"] = eventf2.Hex()
	orderedEvents = append(orderedEvents, eventf2)

	eventf01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[3].Hex(), nodes[1].Events[2].Hex()}, //f0 and f1
		nodes[0].Pub)
	eventf01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventf01)
	index["f01"] = eventf01.Hex()
	orderedEvents = append(orderedEvents, eventf01)

	eventf20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[2].Hex(), nodes[0].Events[3].Hex()}, //f2 and f0
		nodes[2].Pub)
	eventf20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventf20)
	index["f20"] = eventf20.Hex()
	orderedEvents = append(orderedEvents, eventf20)

	eventf02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[4].Hex(), nodes[2].Events[2].Hex()}, //f01 and f2
		nodes[0].Pub)
	eventf02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventf02)
	index["f02"] = eventf02.Hex()
	orderedEvents = append(orderedEvents, eventf02)

	eventf12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[2].Hex(), nodes[2].Events[3].Hex()}, //f1 and f20
		nodes[1].Pub)
	eventf12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventf12)
	index["f12"] = eventf12.Hex()
	orderedEvents = append(orderedEvents, eventf12)

	eventg0 := NewEvent([][]byte{},
		[]string{nodes[0].Events[5].Hex(), nodes[1].Events[3].Hex()}, //f02 and f12
		nodes[0].Pub)
	eventg0.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventg0)
	index["g0"] = eventg0.Hex()
	orderedEvents = append(orderedEvents, eventg0)

	eventg1 := NewEvent([][]byte{},
		[]string{nodes[1].Events[3].Hex(), nodes[0].Events[5].Hex()}, //f12 and f02
		nodes[1].Pub)
	eventg1.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventg1)
	index["g1"] = eventg1.Hex()
	orderedEvents = append(orderedEvents, eventg1)

	eventg2 := NewEvent([][]byte{},
		[]string{nodes[2].Events[3].Hex(), nodes[1].Events[4].Hex()}, //f20 and g1
		nodes[2].Pub)
	eventg2.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventg2)
	index["g2"] = eventg2.Hex()
	orderedEvents = append(orderedEvents, eventg2)

	eventg01 := NewEvent([][]byte{},
		[]string{nodes[0].Events[6].Hex(), nodes[1].Events[4].Hex()}, //g0 and g1
		nodes[0].Pub)
	eventg01.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventg01)
	index["g01"] = eventg01.Hex()
	orderedEvents = append(orderedEvents, eventg01)

	eventg20 := NewEvent([][]byte{},
		[]string{nodes[2].Events[4].Hex(), nodes[0].Events[7].Hex()}, //g2 and g0
		nodes[2].Pub)
	eventg20.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventg20)
	index["g20"] = eventg20.Hex()
	orderedEvents = append(orderedEvents, eventg20)

	eventg02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[7].Hex(), nodes[2].Events[4].Hex()}, //g01 and g2
		nodes[0].Pub)
	eventg02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventg02)
	index["g02"] = eventg02.Hex()
	orderedEvents = append(orderedEvents, eventg02)

	eventg12 := NewEvent([][]byte{},
		[]string{nodes[1].Events[4].Hex(), nodes[2].Events[5].Hex()}, //g1 and g20
		nodes[1].Pub)
	eventg12.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventg12)
	index["g12"] = eventg12.Hex()
	orderedEvents = append(orderedEvents, eventg12)

	eventh0 := NewEvent([][]byte{},
		[]string{nodes[0].Events[8].Hex(), nodes[1].Events[5].Hex()}, //g02 and g12
		nodes[0].Pub)
	eventh0.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, eventh0)
	index["h0"] = eventh0.Hex()
	orderedEvents = append(orderedEvents, eventh0)

	eventh1 := NewEvent([][]byte{},
		[]string{nodes[1].Events[5].Hex(), nodes[0].Events[8].Hex()}, //g12 and g02
		nodes[1].Pub)
	eventh1.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, eventh1)
	index["h1"] = eventh1.Hex()
	orderedEvents = append(orderedEvents, eventh1)

	eventh2 := NewEvent([][]byte{},
		[]string{nodes[2].Events[5].Hex(), nodes[1].Events[6].Hex()}, //g20 and h1
		nodes[2].Pub)
	eventh2.Sign(nodes[2].Key)
	nodes[2].Events = append(nodes[2].Events, eventh2)
	index["h2"] = eventh2.Hex()
	orderedEvents = append(orderedEvents, eventh2)

	//add other nodes

	participants := []string{}
	for _, node := range nodes {
		participants = append(participants, node.PubHex)
	}

	hashgraph := NewHashgraph(participants)
	for i, ev := range orderedEvents {
		if err := hashgraph.InsertEvent(ev); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}
	return hashgraph, index
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

	if l := len(h.Consensus); l != 7 {
		t.Fatalf("length of consensus should be 7 not %d", l)
	}

	if n := getName(index, h.Consensus[0]); n != "e1" {
		t.Fatalf("1st element should be e1 not %s", n)
	}

	//e01 and e0 have the same consensusTimestamp. Their whitened signatures might change with every test run
	//so we need to account for both possible outcomes
	n, m := getName(index, h.Consensus[1]), getName(index, h.Consensus[2])
	if !((n == "e01" && m == "e0") || (n == "e0" && m == "e01")) {
		t.Fatalf("2nd and 3rd elements should be e01 and e0 in whichever order, not %s and %s", n, m)
	}
	n, m = getName(index, h.Consensus[3]), getName(index, h.Consensus[4])
	if !((n == "e2" && m == "e20") || (n == "e20" && m == "e2")) {
		t.Fatalf("4th and 5th elements should be e2 and e20 in whichever order, not %s and %s", n, m)
	}
	if n := getName(index, h.Consensus[5]); n != "e02" {
		t.Fatalf("6th element should be e02 not %s", n)
	}
	if n := getName(index, h.Consensus[6]); n != "e12" {
		t.Fatalf("7th element should be e12 not %s", n)
	}
}

func TestHeight(t *testing.T) {
	h, _ := initConsensusHashgraph()

	if l := len(h.ParticipantEvents[h.Participants[0]]); l != 10 {
		t.Fatalf("0 should have 10 events, not %d", l)
	}
	if l := len(h.ParticipantEvents[h.Participants[1]]); l != 7 {
		t.Fatalf("1 should have 10 events, not %d", l)
	}
	if l := len(h.ParticipantEvents[h.Participants[2]]); l != 7 {
		t.Fatalf("2 should have 10 events, not %d", l)
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
