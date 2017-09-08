package node

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"bitbucket.org/mosaicnet/babble/common"
	"bitbucket.org/mosaicnet/babble/crypto"
	hg "bitbucket.org/mosaicnet/babble/hashgraph"
)

func TestInit(t *testing.T) {
	key, _ := crypto.GenerateECDSAKey()
	participants := map[string]int{
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)): 0,
	}
	core := NewCore(0, key, participants, hg.NewInmemStore(participants, 10), nil, common.NewTestLogger(t))
	if err := core.Init(); err != nil {
		t.Fatalf("Init returned and error: %s", err)
	}
}

func initCores(n int, t *testing.T) ([]Core, []*ecdsa.PrivateKey, map[string]string) {
	cacheSize := 1000

	cores := []Core{}
	index := make(map[string]string)

	participantKeys := []*ecdsa.PrivateKey{}
	participants := make(map[string]int)
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		participantKeys = append(participantKeys, key)
		participants[fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey))] = i
	}

	for i := 0; i < n; i++ {
		core := NewCore(i, participantKeys[i], participants,
			hg.NewInmemStore(participants, cacheSize), nil, common.NewTestLogger(t))
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
			if err := cores[participant].InsertEvent(event, true); err != nil {
				fmt.Printf("error inserting %s: %s\n", getName(index, event.Hex()), err)
			}
		}
	}

	event01 := hg.NewEvent([][]byte{},
		[]string{index["e0"], index["e1"]}, //e0 and e1
		cores[0].PubKey(), 1)
	if err := insertEvent(cores, keys, index, event01, "e01", participant, 0); err != nil {
		fmt.Printf("error inserting e01: %s\n", err)
	}

	event20 := hg.NewEvent([][]byte{},
		[]string{index["e2"], index["e01"]}, //e2 and e01
		cores[2].PubKey(), 1)
	if err := insertEvent(cores, keys, index, event20, "e20", participant, 2); err != nil {
		fmt.Printf("error inserting e20: %s\n", err)
	}

	event12 := hg.NewEvent([][]byte{},
		[]string{index["e1"], index["e20"]}, //e1 and e20
		cores[1].PubKey(), 1)
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
		if err := cores[particant].InsertEvent(event, true); err != nil {
			return err
		}
		index[name] = event.Hex()
	}
	return nil
}

func TestDiff(t *testing.T) {
	cores, keys, index := initCores(3, t)

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
	unknownBy1, err := cores[0].Diff(knownBy1)
	if err != nil {
		t.Fatal(err)
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
	cores, _, index := initCores(3, t)

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
	if k := knownBy0[cores[0].ID()]; k != 1 {
		t.Fatalf("core 0 should have last-index 1 for core 0, not %d", k)
	}
	if k := knownBy0[cores[1].ID()]; k != 0 {
		t.Fatalf("core 0 should have last-index 0 for core 1, not %d", k)
	}
	if k := knownBy0[cores[2].ID()]; k != -1 {
		t.Fatalf("core 0 should have last-index -1 for core 2, not %d", k)
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
	if k := knownBy2[cores[0].ID()]; k != 1 {
		t.Fatalf("core 2 should have last-index 1 for core 0, not %d", k)
	}
	if k := knownBy2[cores[1].ID()]; k != 0 {
		t.Fatalf("core 2 should have last-index 0 core 1, not %d", k)
	}
	if k := knownBy2[cores[2].ID()]; k != 1 {
		t.Fatalf("core 2 should have last-index 1 for core 2, not %d", k)
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
	if k := knownBy1[cores[0].ID()]; k != 1 {
		t.Fatalf("core 1 should have last-index 1 for core 0, not %d", k)
	}
	if k := knownBy1[cores[1].ID()]; k != 1 {
		t.Fatalf("core 1 should have last-index 1 for core 1, not %d", k)
	}
	if k := knownBy1[cores[2].ID()]; k != 1 {
		t.Fatalf("core 1 should have last-index 1 for core 2, not %d", k)
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
|  /|   |--------------------
g02 |   | R2
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
|  /|   |--------------------
f02 |   | R1
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
|  /|   |--------------------
e02 |   | R0 Consensus
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

func initConsensusHashgraph(t *testing.T) []Core {
	cores, _, _ := initCores(3, t)
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
	return cores
}

func synchronizeCores(cores []Core, from int, to int, payload [][]byte) error {
	knownByTo := cores[to].Known()
	unknownByTo, err := cores[from].Diff(knownByTo)
	if err != nil {
		return err
	}

	unknownWire, err := cores[from].ToWire(unknownByTo)
	if err != nil {
		return err
	}

	cores[to].AddTransactions(payload)

	return cores[to].Sync(unknownWire)
}

func syncAndRunConsensus(cores []Core, from int, to int, payload [][]byte) error {
	if err := synchronizeCores(cores, from, to, payload); err != nil {
		return err
	}
	cores[to].RunConsensus()
	return nil
}
func TestConsensus(t *testing.T) {
	cores := initConsensusHashgraph(t)

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

func TestOverSyncLimit(t *testing.T) {
	cores := initConsensusHashgraph(t)

	known := map[int]int{}

	syncLimit := 10

	//positive
	for i := 0; i < 3; i++ {
		known[i] = 1
	}
	if !cores[0].OverSyncLimit(known, syncLimit) {
		t.Fatalf("OverSyncLimit(%v, %v) should return true", known, syncLimit)
	}

	//negative
	for i := 0; i < 3; i++ {
		known[i] = 6
	}
	if cores[0].OverSyncLimit(known, syncLimit) {
		t.Fatalf("OverSyncLimit(%v, %v) should return false", known, syncLimit)
	}

	//edge
	known = map[int]int{
		0: 2,
		1: 3,
		2: 3,
	}
	if cores[0].OverSyncLimit(known, syncLimit) {
		t.Fatalf("OverSyncLimit(%v, %v) should return false", known, syncLimit)
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
