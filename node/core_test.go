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

	"github.com/arrivets/babble/crypto"
	hg "github.com/arrivets/babble/hashgraph"
)

func TestInit(t *testing.T) {
	key, _ := crypto.GenerateECDSAKey()
	participants := []string{
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
	}
	core := NewCore(key, participants, nil)
	if err := core.Init(); err != nil {
		t.Fatalf("Init returned and error: %s", err)
	}
}

func initCores() ([]Core, []*ecdsa.PrivateKey, map[string]string) {
	n := 3
	cores := []Core{}
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
		core := NewCore(participantKeys[i], participantPubs, nil)
		core.Init()
		cores = append(cores, core)
		index[fmt.Sprintf("e%d", i)] = core.Head
	}

	return cores, participantKeys, index
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
func initHashgraph(cores []Core, keys []*ecdsa.PrivateKey, index map[string]string, participant int) {

	for i := 0; i < len(cores); i++ {
		if i != participant {
			event, _ := cores[i].GetEvent(index[fmt.Sprintf("e%d", i)])
			if err := cores[participant].InsertEvent(event); err != nil {
				fmt.Printf("error inserting %s: %s\n", getName(index, event.Hex()), err)
			}
		}
	}

	event01 := hg.NewEvent([][]byte{},
		[]string{index["e0"], index["e1"]}, //e0 and e1
		cores[0].PubKey())
	if err := insertEvent(cores, keys, index, event01, "e01", participant, 0); err != nil {
		fmt.Printf("error inserting e01: %s\n", err)
	}

	event20 := hg.NewEvent([][]byte{},
		[]string{index["e2"], index["e01"]}, //e2 and e01
		cores[2].PubKey())
	if err := insertEvent(cores, keys, index, event20, "e20", participant, 2); err != nil {
		fmt.Printf("error inserting e20: %s\n", err)
	}

	event12 := hg.NewEvent([][]byte{},
		[]string{index["e1"], index["e20"]}, //e1 and e20
		cores[1].PubKey())
	if err := insertEvent(cores, keys, index, event12, "e12", participant, 1); err != nil {
		fmt.Printf("error inserting e12: %s\n", err)
	}
}

func insertEvent(cores []Core, keys []*ecdsa.PrivateKey, index map[string]string,
	event hg.Event, name string, particant int, creator int) error {

	if particant == creator {
		if err := cores[particant].SignAndInsertSelfEvent(event); err != nil {
			return err
		}
		//event is not signed because passed by value
		index[name] = cores[particant].Head
	} else {
		event.Sign(keys[creator])
		if err := cores[particant].InsertEvent(event); err != nil {
			return err
		}
		index[name] = event.Hex()
	}
	return nil
}

func TestDiff(t *testing.T) {
	cores, keys, index := initCores()

	initHashgraph(cores, keys, index, 0)

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

	knownBy1 := cores[1].Known()
	head, unknownBy1 := cores[0].Diff(knownBy1)
	if head != index["e01"] {
		t.Fatalf("head of core 0 should be e01")
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
	cores, _, index := initCores()

	/*
	   core 0           core 1          core 2

	   e0  |   |        |   e1  |       |   |   e2
	   0   1   2        0   1   2       0   1   2
	*/

	//core 1 is going to tell core 0 everything it knows
	if err := synchronizeCores(cores, 1, 0, [][]byte{}); err != nil {
		t.Fatal(err)
	}

	/*
	   core 0           core 1          core 2

	   e01 |   |
	   | \ |   |
	   e0  e1  |        |   e1  |       |   |   e2
	   0   1   2        0   1   2       0   1   2
	*/

	knownBy0 := cores[0].Known()
	if k := knownBy0[fmt.Sprintf("0x%X", cores[0].PubKey())]; k != 2 {
		t.Fatalf("core 0 should have 2 events for core 0, not %d", k)
	}
	if k := knownBy0[fmt.Sprintf("0x%X", cores[1].PubKey())]; k != 1 {
		t.Fatalf("core 0 should have 1 events for core 1, not %d", k)
	}
	if k := knownBy0[fmt.Sprintf("0x%X", cores[2].PubKey())]; k != 0 {
		t.Fatalf("core 0 should have 0 events for core 2, not %d", k)
	}
	core0Head, _ := cores[0].GetHead()
	if core0Head.SelfParent() != index["e0"] {
		t.Fatalf("core 0 head self-parent should be e0")
	}
	if core0Head.OtherParent() != index["e1"] {
		t.Fatalf("core 0 head other-parent should be e1")
	}
	index["e01"] = core0Head.Hex()

	//core 0 is going to tell core 2 everything it knows
	if err := synchronizeCores(cores, 0, 2, [][]byte{}); err != nil {
		t.Fatal(err)
	}

	/*

	   core 0           core 1          core 2

	                                    |   |  e20
	                                    |   | / |
	                                    |   /   |
	                                    | / |   |
	   e01 |   |                        e01 |   |
	   | \ |   |                        | \ |   |
	   e0  e1  |        |   e1  |       e0  e1  e2
	   0   1   2        0   1   2       0   1   2
	*/

	knownBy2 := cores[2].Known()
	if k := knownBy2[fmt.Sprintf("0x%X", cores[0].PubKey())]; k != 2 {
		t.Fatalf("core 2 should have 2 events for core 0, not %d", k)
	}
	if k := knownBy2[fmt.Sprintf("0x%X", cores[1].PubKey())]; k != 1 {
		t.Fatalf("core 2 should have 1 events for core 1, not %d", k)
	}
	if k := knownBy2[fmt.Sprintf("0x%X", cores[2].PubKey())]; k != 2 {
		t.Fatalf("core 2 should have 2 events for core 2, not %d", k)
	}
	core2Head, _ := cores[2].GetHead()
	if core2Head.SelfParent() != index["e2"] {
		t.Fatalf("core 2 head self-parent should be e2")
	}
	if core2Head.OtherParent() != index["e01"] {
		t.Fatalf("core 2 head other-parent should be e01")
	}
	index["e20"] = core2Head.Hex()

	//core 2 is going to tell core 1 everything it knows
	if err := synchronizeCores(cores, 2, 1, [][]byte{}); err != nil {
		t.Fatal(err)
	}

	/*

	   core 0           core 1          core 2

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

	knownBy1 := cores[1].Known()
	if k := knownBy1[fmt.Sprintf("0x%X", cores[0].PubKey())]; k != 2 {
		t.Fatalf("core 1 should have 2 events for core 0, not %d", k)
	}
	if k := knownBy1[fmt.Sprintf("0x%X", cores[1].PubKey())]; k != 2 {
		t.Fatalf("core 1 should have 2 events for core 1, not %d", k)
	}
	if k := knownBy1[fmt.Sprintf("0x%X", cores[2].PubKey())]; k != 2 {
		t.Fatalf("core 1 should have 2 events for core 2, not %d", k)
	}
	core1Head, _ := cores[1].GetHead()
	if core1Head.SelfParent() != index["e1"] {
		t.Fatalf("core 1 head self-parent should be e1")
	}
	if core1Head.OtherParent() != index["e20"] {
		t.Fatalf("core 1 head other-parent should be e20")
	}
	index["e12"] = core1Head.Hex()

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
type play struct {
	from    int
	to      int
	payload [][]byte
}

func TestConsensus(t *testing.T) {
	cores, _, _ := initCores()

	playbook := []play{
		play{from: 0, to: 1, payload: [][]byte{[]byte("e10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("e21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("e02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("f1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("f0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f2")}},

		play{from: 0, to: 1, payload: [][]byte{[]byte("f10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("f02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("g1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("g0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g2")}},

		play{from: 0, to: 1, payload: [][]byte{[]byte("g10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("g02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("h1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("h0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("h2")}},
	}

	for _, play := range playbook {
		if err := syncAndRunConsensus(cores, play.from, play.to, play.payload); err != nil {
			t.Fatal(err)
		}
	}

	if l := len(cores[0].GetConsensusEvents()); l != 6 {
		t.Fatalf("length of consensus should be 6 not %d", l)
	}

	core0Consensus := cores[0].GetConsensusEvents()
	core1Consensus := cores[1].GetConsensusEvents()
	core2Consensus := cores[2].GetConsensusEvents()

	for i, e := range core0Consensus {
		if core1Consensus[i] != e {
			t.Fatalf("core 1 consensus[%d] does not match core 0's", i)
		}
		if core2Consensus[i] != e {
			t.Fatalf("core 2 consensus[%d] does not match core 0's", i)
		}
	}
}

func synchronizeCores(cores []Core, from int, to int, payload [][]byte) error {
	knownByTo := cores[to].Known()
	toHead, unknownByTo := cores[from].Diff(knownByTo)
	return cores[to].Sync(toHead, unknownByTo, payload)
}

func syncAndRunConsensus(cores []Core, from int, to int, payload [][]byte) error {
	if err := synchronizeCores(cores, from, to, payload); err != nil {
		return err
	}
	cores[to].RunConsensus()
	return nil
}

func getName(index map[string]string, hash string) string {
	for name, h := range index {
		if h == hash {
			return name
		}
	}
	return ""
}
