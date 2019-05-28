package node

import (
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/proxy"
)

func initCores(n int, t *testing.T) ([]*Core, map[uint32]*ecdsa.PrivateKey, map[string]string) {
	cacheSize := 1000

	cores := []*Core{}
	index := make(map[string]string)
	participantKeys := map[uint32]*ecdsa.PrivateKey{}
	pirs := []*peers.Peer{}

	for i := 0; i < n; i++ {
		key, _ := keys.GenerateECDSAKey()
		peer := peers.NewPeer(keys.PublicKeyHex(&key.PublicKey), "", "")
		pirs = append(pirs, peer)
		participantKeys[peer.ID()] = key
	}

	peerSet := peers.NewPeerSet(pirs)

	for i, peer := range peerSet.Peers {
		key, _ := participantKeys[peer.ID()]

		core := NewCore(
			NewValidator(key, peer.Moniker),
			peerSet,
			hg.NewInmemStore(cacheSize),
			proxy.DummyCommitCallback,
			common.NewTestLogger(t))

		//Create and save the first Event
		initialEvent := hg.NewEvent([][]byte(nil),
			[]hg.InternalTransaction{},
			nil,
			[]string{"", ""},
			core.validator.PublicKeyBytes(),
			0)

		err := core.SignAndInsertSelfEvent(initialEvent)
		if err != nil {
			t.Fatal(err)
		}

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
func initHashgraph(cores []*Core, keys map[uint32]*ecdsa.PrivateKey, index map[string]string, participant uint32) {
	for i := 0; i < len(cores); i++ {
		if uint32(i) != participant {
			event, _ := cores[i].GetEvent(index[fmt.Sprintf("e%d", i)])
			if err := cores[participant].InsertEventAndRunConsensus(event, true); err != nil {
				fmt.Printf("error inserting %s: %s\n", getName(index, event.Hex()), err)
			}
		}
	}

	event01 := hg.NewEvent([][]byte{},
		[]hg.InternalTransaction{},
		nil,
		[]string{index["e0"], index["e1"]}, //e0 and e1
		cores[0].validator.PublicKeyBytes(), 1)
	if err := insertEvent(cores, keys, index, event01, "e01", participant, cores[0].validator.ID()); err != nil {
		fmt.Printf("error inserting e01: %s\n", err)
	}

	event20 := hg.NewEvent([][]byte{},
		[]hg.InternalTransaction{},
		nil,
		[]string{index["e2"], index["e01"]}, //e2 and e01
		cores[2].validator.PublicKeyBytes(), 1)
	if err := insertEvent(cores, keys, index, event20, "e20", participant, cores[2].validator.ID()); err != nil {
		fmt.Printf("error inserting e20: %s\n", err)
	}

	event12 := hg.NewEvent([][]byte{},
		[]hg.InternalTransaction{},
		nil,
		[]string{index["e1"], index["e20"]}, //e1 and e20
		cores[1].validator.PublicKeyBytes(), 1)
	if err := insertEvent(cores, keys, index, event12, "e12", participant, cores[1].validator.ID()); err != nil {
		fmt.Printf("error inserting e12: %s\n", err)
	}
}

func insertEvent(cores []*Core, keys map[uint32]*ecdsa.PrivateKey, index map[string]string,
	event *hg.Event, name string, particant uint32, creator uint32) error {

	if particant == creator {
		if err := cores[particant].SignAndInsertSelfEvent(event); err != nil {
			return err
		}
		//event is not signed because passed by value
		index[name] = cores[particant].Head
	} else {
		event.Sign(keys[creator])
		if err := cores[particant].InsertEventAndRunConsensus(event, true); err != nil {
			return err
		}
		index[name] = event.Hex()
	}
	return nil
}

func TestEventDiff(t *testing.T) {
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

	knownBy1 := cores[1].KnownEvents()
	unknownBy1, err := cores[0].EventDiff(knownBy1)
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
	if err := synchronizeCores(cores, 1, 0, [][]byte{}, []hg.InternalTransaction{}); err != nil {
		t.Fatal(err)
	}

	/*
	   core 0           core 1          core 2

	   e01 |   |
	   | \ |   |
	   e0  e1  |        |   e1  |       |   |   e2
	   0   1   2        0   1   2       0   1   2
	*/

	knownBy0 := cores[0].KnownEvents()
	if k := knownBy0[cores[0].validator.ID()]; k != 1 {
		t.Fatalf("core 0 should have last-index 1 for core 0, not %d", k)
	}
	if k := knownBy0[cores[1].validator.ID()]; k != 0 {
		t.Fatalf("core 0 should have last-index 0 for core 1, not %d", k)
	}
	if k := knownBy0[cores[2].validator.ID()]; k != -1 {
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
	if err := synchronizeCores(cores, 0, 2, [][]byte{}, []hg.InternalTransaction{}); err != nil {
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

	knownBy2 := cores[2].KnownEvents()
	if k := knownBy2[cores[0].validator.ID()]; k != 1 {
		t.Fatalf("core 2 should have last-index 1 for core 0, not %d", k)
	}
	if k := knownBy2[cores[1].validator.ID()]; k != 0 {
		t.Fatalf("core 2 should have last-index 0 core 1, not %d", k)
	}
	if k := knownBy2[cores[2].validator.ID()]; k != 1 {
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
	if err := synchronizeCores(cores, 2, 1, [][]byte{}, []hg.InternalTransaction{}); err != nil {
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

	knownBy1 := cores[1].KnownEvents()
	if k := knownBy1[cores[0].validator.ID()]; k != 1 {
		t.Fatalf("core 1 should have last-index 1 for core 0, not %d", k)
	}
	if k := knownBy1[cores[1].validator.ID()]; k != 1 {
		t.Fatalf("core 1 should have last-index 1 for core 1, not %d", k)
	}
	if k := knownBy1[cores[2].validator.ID()]; k != 1 {
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
	from        int
	to          int
	payload     [][]byte
	internalTxs []hg.InternalTransaction
}

func initConsensusHashgraph(t *testing.T) []*Core {
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
		if err := syncAndRunConsensus(cores, play.from, play.to, play.payload, play.internalTxs); err != nil {
			t.Fatal(err)
		}
	}
	return cores
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

	//positive
	known := map[uint32]int{
		cores[0].validator.ID(): 1,
		cores[1].validator.ID(): 1,
		cores[2].validator.ID(): 1,
	}

	syncLimit := 10
	enableSyncLimit := true

	if !cores[0].OverSyncLimit(known, syncLimit, enableSyncLimit) {
		t.Fatalf("OverSyncLimit(%v, %v) should return true", known, syncLimit)
	}

	//negative
	known = map[uint32]int{
		cores[0].validator.ID(): 6,
		cores[1].validator.ID(): 6,
		cores[2].validator.ID(): 6,
	}

	if cores[0].OverSyncLimit(known, syncLimit, enableSyncLimit) {
		t.Fatalf("OverSyncLimit(%v, %v) should return false", known, syncLimit)
	}

	//edge
	known = map[uint32]int{
		cores[0].validator.ID(): 2,
		cores[1].validator.ID(): 3,
		cores[2].validator.ID(): 3,
	}
	if cores[0].OverSyncLimit(known, syncLimit, enableSyncLimit) {
		t.Fatalf("OverSyncLimit(%v, %v) should return false", known, syncLimit)
	}

}

/*
    |   |   |   |-----------------
	|   w31 |   | R3
	|	| \ |   |
    |   |  w32  |
    |   |   | \ |
    |   |   |  w33
    |   |   | / |-----------------
    |   |  g21  | R2
	|   | / |   |
	|  w21  |   |
	|	| \ |   |
    |   |  w22  |
    |   |   | \ |
    |   |   |  w23
    |   |   | / |-----------------
    |   |  f21  | R1
	|   | / |   | LastConsensusRound
	|  w11  |   |
	|	| \ |   |
    |   |   \   |
    |   |   | \ |
	|   |   |  w13
	|   |   | / |
   FSE  |  w12  | FSE is only added after FastForward
    |\  | / |   | -----------------
    |  e13  |   | R0
	|	| \ |   |
    |   |   \   |
    |   |   | \ |
    |   |   |  e32
    |   |   | / |
    |   |  e21  | All Events in Round 0 are Consensus Events.
    |   | / |   |
    |   e1  e2  e3
    0	1	2	3
*/
func initFFHashgraph(cores []*Core, t *testing.T) {
	playbook := []play{
		play{from: 1, to: 2, payload: [][]byte{[]byte("e21")}},
		play{from: 2, to: 3, payload: [][]byte{[]byte("e32")}},
		play{from: 3, to: 1, payload: [][]byte{[]byte("e13")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w12")}},
		play{from: 2, to: 3, payload: [][]byte{[]byte("w13")}},
		play{from: 3, to: 1, payload: [][]byte{[]byte("w11")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f21")}},
		play{from: 2, to: 3, payload: [][]byte{[]byte("w23")}},
		play{from: 3, to: 2, payload: [][]byte{[]byte("w22")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("w21")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g21")}},
		play{from: 2, to: 3, payload: [][]byte{[]byte("w33")}},
		play{from: 3, to: 2, payload: [][]byte{[]byte("w32")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("w31")}},
	}

	for k, play := range playbook {
		if err := syncAndRunConsensus(cores, play.from, play.to, play.payload, play.internalTxs); err != nil {
			t.Fatalf("play %d: %s", k, err)
		}
	}
}

func TestConsensusFF(t *testing.T) {
	cores, _, _ := initCores(4, t)
	initFFHashgraph(cores, t)

	if r := cores[1].GetLastConsensusRoundIndex(); r == nil || *r != 1 {
		disp := "nil"
		if r != nil {
			disp = strconv.Itoa(*r)
		}
		t.Fatalf("Cores[1] last consensus Round should be 1, not %s", disp)
	}

	if l := len(cores[1].GetConsensusEvents()); l != 6 {
		t.Fatalf("Node 1 should have 6 consensus events, not %d", l)
	}

	core1Consensus := cores[1].GetConsensusEvents()
	core2Consensus := cores[2].GetConsensusEvents()
	core3Consensus := cores[3].GetConsensusEvents()

	for i, e := range core1Consensus {
		if core2Consensus[i] != e {
			t.Fatalf("Node 2 consensus[%d] does not match Node 1's", i)
		}
		if core3Consensus[i] != e {
			t.Fatalf("Node 3 consensus[%d] does not match Node 1's", i)
		}
	}
}

func TestCoreFastForward(t *testing.T) {
	cores, _, _ := initCores(4, t)
	initFFHashgraph(cores, t)

	t.Run("Test no Anchor", func(t *testing.T) {
		//Test no anchor block
		_, _, err := cores[1].GetAnchorBlockWithFrame()
		if err == nil {
			t.Fatal("GetAnchorBlockWithFrame should throw an error because there is no anchor block yet")
		}
	})

	block0, err := cores[1].hg.Store.GetBlock(0)
	if err != nil {
		t.Fatal(err)
	}

	//collect signatures
	signatures := make([]hg.BlockSignature, 3)
	for k, c := range cores[1:] {
		b, err := c.hg.Store.GetBlock(0)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := c.SignBlock(b)
		if err != nil {
			t.Fatal(err)
		}
		signatures[k] = sig
	}

	t.Run("Test not enough signatures", func(t *testing.T) {
		//Append only 1 signatures
		if err := block0.SetSignature(signatures[0]); err != nil {
			t.Fatal(err)
		}

		//Save Block
		if err := cores[1].hg.Store.SetBlock(block0); err != nil {
			t.Fatal(err)
		}
		//Assign AnchorBlock
		cores[1].hg.AnchorBlock = new(int)
		*cores[1].hg.AnchorBlock = 0

		//Now the function should find an AnchorBlock
		block, frame, err := cores[1].GetAnchorBlockWithFrame()
		if err != nil {
			t.Fatal(err)
		}

		err = cores[0].FastForward(cores[1].validator.PublicKeyHex(), block, frame)
		//We should get an error because AnchorBlock doesnt contain enough
		//signatures
		if err == nil {
			t.Fatal("FastForward should throw an error because the Block does not contain enough signatures")
		}
	})

	t.Run("Test positive", func(t *testing.T) {
		//Append the 2nd and 3rd signatures
		for i := 1; i < 3; i++ {
			if err := block0.SetSignature(signatures[i]); err != nil {
				t.Fatal(err)
			}
		}

		//Save Block
		if err := cores[1].hg.Store.SetBlock(block0); err != nil {
			t.Fatal(err)
		}

		block, frame, err := cores[1].GetAnchorBlockWithFrame()
		if err != nil {
			t.Fatal(err)
		}

		//This operation clears the Events' private fields, which need to be
		//recomputed (round, roundReceived, etc).
		marshalledFrame, _ := frame.Marshal()
		unmarshalledFrame := new(hg.Frame)
		unmarshalledFrame.Unmarshal(marshalledFrame)

		err = cores[0].FastForward(cores[1].validator.PublicKeyHex(), block, unmarshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		knownBy0 := cores[0].KnownEvents()
		if err != nil {
			t.Fatal(err)
		}

		expectedKnown := map[uint32]int{
			cores[0].validator.ID(): -1,
			cores[1].validator.ID(): 1,
			cores[2].validator.ID(): 1,
			cores[3].validator.ID(): 1,
		}

		if !reflect.DeepEqual(knownBy0, expectedKnown) {
			t.Fatalf("Cores[0].Known should be %v, not %v", expectedKnown, knownBy0)
		}

		if r := cores[0].GetLastConsensusRoundIndex(); r == nil || *r != 1 {
			t.Fatalf("Cores[0] last consensus Round should be 1, not %v", r)
		}

		if lbi := cores[0].hg.Store.LastBlockIndex(); lbi != 0 {
			t.Fatalf("Cores[0].hg.LastBlockIndex should be 0, not %d", lbi)
		}

		sBlock, err := cores[0].hg.Store.GetBlock(block.Index())
		if err != nil {
			t.Fatalf("Error retrieving latest Block from reset hashgraph: %v", err)
		}
		if !reflect.DeepEqual(sBlock.Body, block.Body) {
			t.Fatalf("Blocks defer")
		}
	})
}

/*
We introduce a JoinRequest at round 1, which is received at round 2, and updates
the PeerSet at round 8 (2 + 6)

            |    | \  |
Round 8     |    |   w82
P:[0,1,2,3] |    | /  |
            |   w81   |
            |  / |    |
            w80  |    |
        -------\----------
            |   l12   |
            |    | \  |
Round 7     |    |   w72
P:[0,1,2]   |    | /  |
            |   w71   |
            |  / |    |
            w70  |    |
        -------\----------
Round 6     |   k12   |
P:[0,1,2]   |    | \  |
            |    |   w62
            |    | /  |
            |   w61   |
            |  / |    |
            w60  |    |
        -------\----------
            |   j12   |
            |    | \  |  AnchorBlock
Round 5     |    |   w52
P:[0,1,2]   |    | /  |
            |   w51   |
            |  / |    |
            w50  |    |
        -------\----------
            |   i12   |
            |    | \  |
Round 4     |    |   w42
P:[0,1,2]   |    | /  |
            |   w41   |
            |  / |    |
            w40  |    |
        -------\----------
            |   h12   |
Round 3     |    | \  |
P:[0,1,2]   |    |   w32
            |    |  / |
            |   w31   |
            | /  |    |
           w30   |    |
            |  \ |    |
        ------------------
Round 2     |    |  \ |
P:[0,1,2]   |    |   g21
            |    | /  |
            |   w21   |
            | /  |    |
           w20   |    |
            |  \ |    |
            |    | \  |
            |    |   w22
        -----------/------
Round 1     |   f10   |
P:[0,1,2]   | /  |    |
           w10   |    |
            |  \ |    |
            |    | \  |
            |    |   w12*
            |    |  / |
            |   w11   |
         -----/------------
Round 0	   e12   |    |   Block 0
P:[0,1,2]   |  \ |    |
            |    | \  |
            |    |   e21
            |    | /  |
            |   e10   |
            |  / |    |
           w00  w01  w02
            |    |    |
            R0   R1   R2
            0    1    2
*/
func initR2DynHashgraph(t *testing.T) (cores []*Core, bobPeer *peers.Peer, bobKey *ecdsa.PrivateKey) {
	//Initialize first 3 cores. They will have the same PeerSet of 3.
	cores, _, _ = initCores(3, t)

	//Initialize the joining Peer (bob)
	bobKey, _ = keys.GenerateECDSAKey()
	bobPubHex := keys.PublicKeyHex(&bobKey.PublicKey)
	bobPeer = peers.NewPeer(bobPubHex, "", "")

	//Initialize internalTx
	itx := hg.NewInternalTransactionJoin(*bobPeer)
	itx.Sign(bobKey)

	//Insert a JoinRequest in a Round0 Event
	playbook := []play{
		play{from: 0, to: 1, payload: [][]byte{[]byte("e10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("e21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("e12")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w11")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w12")},
			internalTxs: []hg.InternalTransaction{itx}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("w10")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("f10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w22")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("w20")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w21")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("w30")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w31")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w32")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("h12")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("w40")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w41")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w42")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("i12")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("w50")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w51")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w52")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("j12")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("w60")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w61")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w62")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("k12")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("w70")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w71")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w72")}},
		play{from: 2, to: 1, payload: [][]byte{[]byte("l12")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("w80")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("w81")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("w82")}},
	}

	for k, play := range playbook {
		if err := syncAndRunConsensus(cores, play.from, play.to, play.payload, play.internalTxs); err != nil {
			t.Fatalf("play %d: %s", k, err)
		}
	}

	return cores, bobPeer, bobKey
}

func TestR2DynConsensus(t *testing.T) {
	cores, _, _ := initR2DynHashgraph(t)

	for i, c := range cores {
		frame1, err := c.hg.Store.GetFrame(1)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("frame1: %v", frame1)
		if lcr := c.hg.LastConsensusRound; lcr == nil || *lcr != 6 {
			t.Fatalf("cores[%d] LastConsensusRound should be 6, not %d", i, *lcr)
		}
		if ps, _ := c.hg.Store.GetPeerSet(8); ps.Len() != 4 {
			t.Fatalf("cores[%d] PeerSet(8) should contain 4 peers, not %d", i, ps.Len())
		}
	}
}

func TestCoreFastForwardAfterJoin(t *testing.T) {
	cores, bobPeer, bobKey := initR2DynHashgraph(t)

	initPeerSet, err := cores[0].hg.Store.GetPeerSet(0)
	if err != nil {
		t.Fatal(err)
	}

	bobCore := NewCore(
		NewValidator(bobKey, bobPeer.Moniker),
		initPeerSet,
		hg.NewInmemStore(1000),
		proxy.DummyCommitCallback,
		common.NewTestLogger(t))

	bobCore.SetHeadAndSeq()

	cores = append(cores, bobCore)

	/***************************************************************************
		Manually FastForward Bob from cores[2]

		Testing 2 scenarios:

			- AnchorBlock: check that the AnchorBlock (Block 5) is selected
						   correctly and that FuturePeerSets works.

			- Block 0: check that FastForwarding from a Round below the PeerSet
			           change works.
	***************************************************************************/

	type play struct {
		block           *hg.Block
		frame           *hg.Frame
		roundLowerBound int
	}

	plays := []play{}

	//Prepare Block 0 scenario
	block0, err := cores[2].hg.Store.GetBlock(0)
	if err != nil {
		t.Fatal(err)
	}

	frame0, err := cores[2].hg.Store.GetFrame(block0.RoundReceived())
	if err != nil {
		t.Fatal(err)
	}

	plays = append(plays, play{block0, frame0, 0})

	//Prepare AnchorBlock scenario
	anchorBlock, anchorFrame, err := cores[2].hg.GetAnchorBlockWithFrame()
	if err != nil {
		t.Fatal(err)
	}

	plays = append(plays, play{anchorBlock, anchorFrame, 6})

	/***************************************************************************
		Run the same test for both scenarios
	***************************************************************************/

	for _, p := range plays {

		/***********************************************************************
			FastForward, Sync, and Run Consensus
		***********************************************************************/

		marshalledBlock, err := p.block.Marshal()
		if err != nil {
			t.Fatal(err)
		}

		var unmarshalledBlock hg.Block
		err = unmarshalledBlock.Unmarshal(marshalledBlock)
		if err != nil {
			t.Fatal(err)
		}

		marshalledFrame, err := p.frame.Marshal()
		if err != nil {
			t.Fatal(err)
		}

		var unmarshalledFrame hg.Frame
		err = unmarshalledFrame.Unmarshal(marshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		err = cores[3].FastForward(cores[2].validator.PublicKeyHex(), &unmarshalledBlock, &unmarshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		//continue after FastForward
		err = syncAndRunConsensus(cores, 2, 3, [][]byte{}, []hg.InternalTransaction{})
		if err != nil {
			t.Fatal(err)
		}

		/***********************************************************************
			Check Known
		***********************************************************************/

		knownBy3 := cores[3].KnownEvents()
		if err != nil {
			t.Fatal(err)
		}

		expectedKnown := map[uint32]int{
			cores[0].validator.ID(): 9,
			cores[1].validator.ID(): 15,
			cores[2].validator.ID(): 10,
			cores[3].validator.ID(): 0,
		}

		if !reflect.DeepEqual(knownBy3, expectedKnown) {
			t.Fatalf("Cores[3].Known should be %v, not %v", expectedKnown, knownBy3)
		}

		/***********************************************************************
			Check Rounds
		***********************************************************************/

		//The fame of witnesses of the FastForward's Block RoundReceived and
		//below are not reprocessed after Reset. No need to test those rounds.
		for i := p.roundLowerBound; i <= 8; i++ {
			c3RI, err := cores[3].hg.Store.GetRound(i)
			if err != nil {
				t.Fatal(err)
			}

			c2RI, err := cores[2].hg.Store.GetRound(i)
			if err != nil {
				t.Fatal(err)
			}

			c3RIw := c3RI.Witnesses()
			c2RIw := c2RI.Witnesses()
			sort.Strings(c3RIw)
			sort.Strings(c2RIw)

			if !reflect.DeepEqual(c3RIw, c2RIw) {
				t.Logf("Round(%d).Witnesses do not match", i)
			}

			if !reflect.DeepEqual(c3RI.CreatedEvents, c3RI.CreatedEvents) {
				t.Logf("Round(%d).CreatedEvents do not match", i)
			}

			c3RIr := c3RI.ReceivedEvents
			c2RIr := c2RI.ReceivedEvents
			sort.Strings(c3RIr)
			sort.Strings(c2RIr)

			if !reflect.DeepEqual(c3RIw, c2RIw) {
				t.Logf("Round(%d).ReceivedEvents do not match", i)
			}
		}

		/***********************************************************************
			Check PeerSets
		***********************************************************************/

		for i := p.roundLowerBound; i <= 8; i++ {
			c3PS, err := cores[3].hg.Store.GetPeerSet(i)
			if err != nil {
				t.Fatal(err)
			}

			c2PS, err := cores[2].hg.Store.GetPeerSet(i)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(c3PS.Hex(), c2PS.Hex()) {
				t.Fatalf("PeerSet(%d) does not match", i)
			}
		}

		/***********************************************************************
			Check Consensus Rounds and Blocks
		***********************************************************************/

		if r := cores[3].GetLastConsensusRoundIndex(); r == nil || *r != 6 {
			t.Fatalf("Cores[3] last consensus Round should be 4, not %v", *r)
		}

		if lbi := cores[3].hg.Store.LastBlockIndex(); lbi != 5 {
			t.Fatalf("Cores[3].hg.LastBlockIndex should be 5, not %d", lbi)
		}
	}

}

/******************************************************************************/

func synchronizeCores(cores []*Core, from int, to int, payload [][]byte, internalTxs []hg.InternalTransaction) error {
	knownByTo := cores[to].KnownEvents()
	unknownByTo, err := cores[from].EventDiff(knownByTo)
	if err != nil {
		return err
	}

	unknownWire, err := cores[from].ToWire(unknownByTo)
	if err != nil {
		return err
	}

	cores[to].AddTransactions(payload)

	for _, it := range internalTxs {
		cores[to].AddInternalTransaction(it)
	}

	return cores[to].Sync(cores[from].validator.ID(), unknownWire)
}

func syncAndRunConsensus(cores []*Core, from int, to int, payload [][]byte, internalTxs []hg.InternalTransaction) error {
	if err := synchronizeCores(cores, from, to, payload, internalTxs); err != nil {
		return err
	}
	cores[to].ProcessSigPool()
	return nil
}

func getName(index map[string]string, hash string) string {
	for name, h := range index {
		if h == hash {
			return name
		}
	}
	return fmt.Sprintf("%s not found", hash)
}
