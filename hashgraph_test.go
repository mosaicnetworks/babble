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

	if ss, c := h.StronglySee(index["e12"], index["e0"]); !ss {
		t.Fatalf("e12 should strongly see e0. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e12"], index["e1"]); !ss {
		t.Fatalf("e12 should strongly see e1. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e12"], index["e01"]); !ss {
		t.Fatalf("e12 should strongly see e01. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e20"], index["e1"]); !ss {
		t.Fatalf("e20 should strongly see e1. %d sentinels", c)
	}

	//false negatives
	if ss, c := h.StronglySee(index["e12"], index["e2"]); ss {
		t.Fatalf("e12 should not strongly see e2. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e12"], index["e20"]); ss {
		t.Fatalf("e12 should not strongly see e20. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e20"], index["e01"]); ss {
		t.Fatalf("e20 should not strongly see e01. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e20"], index["e0"]); ss {
		t.Fatalf("e20 should not strongly see e0. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e20"], index["e2"]); ss {
		t.Fatalf("e20 should not strongly see e2. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e01"], index["e0"]); ss {
		t.Fatalf("e01 should not strongly see e0. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e01"], index["e1"]); ss {
		t.Fatalf("e01 should not strongly see e1. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e01"], index["e2"]); ss {
		t.Fatalf("e01 should not strongly see e2. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e0"], index["e0"]); ss {
		t.Fatalf("e0 should not strongly see e0. %d sentinels", c)
	}

	//fork
	h, index = initForkHashgraph()
	if ss, c := h.StronglySee(index["e12"], index["a"]); ss {
		t.Fatalf("e12 should not strongly see 'a' because of fork. %d sentinels", c)
	}
	if ss, c := h.StronglySee(index["e12"], index["e2"]); ss {
		t.Fatalf("e12 should not strongly see e2 because of fork. %d sentinels", c)
	}

}

func TestParentRound(t *testing.T) {
	h, index := initHashgraph()
	e01 := h.Events[index["e01"]]
	e01.round = 1
	h.Events[index["e01"]] = e01

	if r := h.ParentRound(index["e0"]); r != 0 {
		t.Fatalf("parent round of e0 should be 0, not %d", r)
	}
	if r := h.ParentRound(index["e1"]); r != 0 {
		t.Fatalf("parent round of e1 should be 0, not %d", r)
	}
	if r := h.ParentRound(index["e01"]); r != 0 {
		t.Fatalf("parent round of e01 should be 0, not %d", r)
	}
	if r := h.ParentRound(index["e20"]); r != 1 {
		t.Fatalf("parent round of e20 should be 1, not %d", r)
	}
}

func TestWitness(t *testing.T) {
	h, index := initHashgraph()
	e01 := h.Events[index["e01"]]
	e01.round = 1
	h.Events[index["e01"]] = e01

	if !h.Witness(index["e0"]) {
		t.Fatalf("e0 should be witness")
	}
	if !h.Witness(index["e1"]) {
		t.Fatalf("e1 should be witness")
	}
	if !h.Witness(index["e2"]) {
		t.Fatalf("e2 should be witness")
	}
	if !h.Witness(index["e01"]) {
		t.Fatalf("e01 should be witness")
	}

	if h.Witness(index["e12"]) {
		t.Fatalf("e12 should not be witness")
	}
	if h.Witness(index["e20"]) {
		t.Fatalf("e20 should not be witness")
	}

}

/*   e10  |
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

	event02 := NewEvent([][]byte{},
		[]string{nodes[0].Events[1].Hex(), nodes[2].Events[0].Hex()}, //e01 and e2
		nodes[0].Pub)
	event02.Sign(nodes[0].Key)
	nodes[0].Events = append(nodes[0].Events, event02)
	index["e02"] = event02.Hex()

	event10 := NewEvent([][]byte{},
		[]string{nodes[1].Events[1].Hex(), nodes[0].Events[2].Hex()}, //e12 and e02
		nodes[1].Pub)
	event10.Sign(nodes[1].Key)
	nodes[1].Events = append(nodes[1].Events, event10)
	index["e10"] = event10.Hex()

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

func TestRoundInc(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := []string{}
	round0Witnesses = append(round0Witnesses, index["e0"])
	round0Witnesses = append(round0Witnesses, index["e1"])
	round0Witnesses = append(round0Witnesses, index["e2"])
	h.RoundWitnesses[0] = round0Witnesses

	if !h.RoundInc(index["e10"]) {
		t.Fatal("RoundInc e10 should be true")
	}

	if h.RoundInc(index["e12"]) {
		t.Fatal("RoundInc e12 should be false because it doesnt strongly see e2")
	}
}

func TestRound(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := []string{}
	round0Witnesses = append(round0Witnesses, index["e0"])
	round0Witnesses = append(round0Witnesses, index["e1"])
	round0Witnesses = append(round0Witnesses, index["e2"])
	h.RoundWitnesses[0] = round0Witnesses

	if r := h.Round(index["e10"]); r != 1 {
		t.Fatalf("round of e10 should be 1 not %d", r)
	}
	if r := h.Round(index["e12"]); r != 0 {
		t.Fatalf("round of e12 should be 0 not %d", r)
	}

}

func TestRoundDiff(t *testing.T) {
	h, index := initRoundHashgraph()

	round0Witnesses := []string{}
	round0Witnesses = append(round0Witnesses, index["e0"])
	round0Witnesses = append(round0Witnesses, index["e1"])
	round0Witnesses = append(round0Witnesses, index["e2"])
	h.RoundWitnesses[0] = round0Witnesses

	if d, err := h.RoundDiff(index["e10"], index["e02"]); d != 1 {
		if err != nil {
			t.Fatalf("RoundDiff(e10, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e10, e02) should be 1 not %d", d)
	}

	if d, err := h.RoundDiff(index["e02"], index["e10"]); d != -1 {
		if err != nil {
			t.Fatalf("RoundDiff(e02, e10) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e02, e10) should be -1 not %d", d)
	}
	if d, err := h.RoundDiff(index["e10"], index["e02"]); d != 1 {
		if err != nil {
			t.Fatalf("RoundDiff(e10, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e10, e02) should be 1 not %d", d)
	}
	if d, err := h.RoundDiff(index["e20"], index["e02"]); d != 0 {
		if err != nil {
			t.Fatalf("RoundDiff(e20, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e20, e02) should be 1 not %d", d)
	}
}
