package hashgraph

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*
We introduce a new participant at Round 2, and remove another participant at
round 5.

Round 7
P: [1,2,3]      w71   |    |
         ----------\--------------
Round 6          |   h23   |
P: [1,2,3]       |    | \  |
                 |    |   w63
                 |    | /  |
				 |  / |    |
                w61   |    |
                 | \  |    |
                 |   w62   |
		 ----------------\--------
Round 5          |    |   j31
P: [1,2,3]       |    | /  |
				 |  / |    |
                w51   |    |
                 | \  |    |
                 |   w52   |
                 |    | \  |
		         |    |   w53
         ---------------/---------
Round 4          |   w42   |
P:[0,1,2,3]      | /  |    |
                w41   |    |
               / |    |    |
      		w40  |    |    |
            |    \    |    |
            |    |    \    |
		    |    |    |   w43
         ---------------/---------
Round 3     |    |   w32   |
P:[0,1,2,3] |    | /  |    |
            |   w31   |    |
            |  / |    |    |
            w30  |    |    |
            |    \    |    |
            |    |    \    |
		    |    |    |   w33
         -------------------------
Round 2		|    |    | /  |
P:[0,1,2,3] |    |   g21   R3
			|    | /  |
			|   w21   |
			| /  |    |
		   w20   |    |
		    |  \ |    |
		    |    | \  |
		    |    |   w22
		 -----------/------
Round 1		|   f10   |
P:[0,1,2]	| /  |    |
		   w10   |    |
		    |  \ |    |
		    |    | \  |
		    |    |   w12
		    |    |  / |
		    |   w11   |
		 -----/------------
Round 0	   e12   |    |
P:[0,1,2]   |  \ |    |
		    |    | \  |
		    |    |   e21
		    |    | /  |
		    |   e10   |
		    |  / |    |
		   w00  w01  w02
			|    |    |
		    R0   R1   R2
			0	 1	  2
*/
func initR2DynHashgraph(t testing.TB) (*Hashgraph, map[string]string) {
	nodes, index, orderedEvents, peerSet := initHashgraphNodes(3)

	for i := range peerSet.Peers {
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, nil, []string{"", ""}, nodes[i].PubBytes, 0)
		nodes[i].signAndAddEvent(event, name, index, orderedEvents)
	}

	plays := []play{
		{1, 1, "w01", "w00", "e10", [][]byte{[]byte("e10")}, nil},
		{2, 1, "w02", "e10", "e21", [][]byte{[]byte("e21")}, nil},
		{0, 1, "w00", "e21", "e12", [][]byte{[]byte("e12")}, nil},
		{1, 2, "e10", "e12", "w11", [][]byte{[]byte("w11")}, nil},
		{2, 2, "e21", "w11", "w12", [][]byte{[]byte("w12")}, nil},
		{0, 2, "e12", "w12", "w10", [][]byte{[]byte("w10")}, nil},
		{1, 3, "w11", "w10", "f10", [][]byte{[]byte("f10")}, nil},
		{2, 3, "w12", "f10", "w22", [][]byte{[]byte("w22")}, nil},
		{0, 3, "w10", "w22", "w20", [][]byte{[]byte("w20")}, nil},
		{1, 4, "f10", "w20", "w21", [][]byte{[]byte("w21")}, nil},
		{2, 4, "w22", "w21", "g21", [][]byte{[]byte("g21")}, nil},
	}

	playEvents(plays, nodes, index, orderedEvents)

	hg := createHashgraph(false, orderedEvents, peerSet, t)

	/***************************************************************************
		Add Participant 3; new Peerset for Round2
	***************************************************************************/

	//create new node
	key3, _ := keys.GenerateECDSAKey()
	node3 := NewTestNode(key3)
	nodes = append(nodes, node3)
	peer3 := peers.NewPeer(node3.PubHex, "", "")
	index["R3"] = ""
	newPeerSet := peerSet.WithNewPeer(peer3)

	//Set Round 2 PeerSet
	err := hg.Store.SetPeerSet(2, newPeerSet)
	if err != nil {
		t.Fatal(err)
	}

	/***************************************************************************
		Continue inserting Events with new participant
	***************************************************************************/

	plays = []play{
		{3, 0, "R3", "g21", "w33", [][]byte{[]byte("w33")}, nil},
		{0, 4, "w20", "w33", "w30", [][]byte{[]byte("w30")}, nil},
		{1, 5, "w21", "w30", "w31", [][]byte{[]byte("w31")}, nil},
		{2, 5, "g21", "w31", "w32", [][]byte{[]byte("w32")}, nil},
		{3, 1, "w33", "w32", "w43", [][]byte{[]byte("w43")}, nil},
		{0, 5, "w30", "w43", "w40", [][]byte{[]byte("w40")}, nil},
		{1, 6, "w31", "w40", "w41", [][]byte{[]byte("w41")}, nil},
		{2, 6, "w32", "w41", "w42", [][]byte{[]byte("w42")}, nil},
	}

	orderedEvents = &[]*Event{}

	playEvents(plays, nodes, index, orderedEvents)

	for i, ev := range *orderedEvents {
		if err := hg.InsertEvent(ev, true); err != nil {
			t.Fatalf("ERROR inserting event %d: %s\n", i, err)
		}
	}

	/***************************************************************************
		Remove Participant 0; new Peerset for Round5
	***************************************************************************/

	newPeerSet2 := newPeerSet.WithRemovedPeer(newPeerSet.Peers[0])

	//Set Round 5 PeerSet
	err = hg.Store.SetPeerSet(5, newPeerSet2)
	if err != nil {
		t.Fatal(err)
	}

	/***************************************************************************
		Continue inserting Events with new participant
	***************************************************************************/

	plays = []play{
		{3, 2, "w43", "w42", "w53", [][]byte{[]byte("w53")}, nil},
		{2, 7, "w42", "w53", "w52", [][]byte{[]byte("w52")}, nil},
		{1, 7, "w41", "w52", "w51", [][]byte{[]byte("w51")}, nil},
		{3, 3, "w53", "w51", "j31", [][]byte{[]byte("j31")}, nil},
		{2, 8, "w52", "j31", "w62", [][]byte{[]byte("w62")}, nil},
		{1, 8, "w51", "w62", "w61", [][]byte{[]byte("w61")}, nil},
		{3, 4, "j31", "w61", "w63", [][]byte{[]byte("w63")}, nil},
		{2, 9, "w62", "w63", "h23", [][]byte{[]byte("h23")}, nil},
		{1, 9, "w61", "h23", "w71", [][]byte{[]byte("w71")}, nil},
	}

	orderedEvents = &[]*Event{}

	playEvents(plays, nodes, index, orderedEvents)

	for i, ev := range *orderedEvents {
		if err := hg.InsertEvent(ev, true); err != nil {
			t.Fatalf("ERROR inserting event %d: %s\n", i, err)
		}
	}

	return hg, index
}

func TestR2DynDivideRounds(t *testing.T) {
	h, index := initR2DynHashgraph(t)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}

	/**************************************************************************/

	//[event] => {lamportTimestamp, round}
	type tr struct {
		t, r int
	}
	expectedTimestamps := map[string]tr{
		"w00": {0, 0},
		"w01": {0, 0},
		"w02": {0, 0},
		"e10": {1, 0},
		"e21": {2, 0},
		"e12": {3, 0},
		"w11": {4, 1},
		"w12": {5, 1},
		"w10": {6, 1},
		"f10": {7, 1},
		"w22": {8, 2},
		"w20": {9, 2},
		"w21": {10, 2},
		"g21": {11, 2},
		"w33": {12, 3},
		"w30": {13, 3},
		"w31": {14, 3},
		"w32": {15, 3},
		"w43": {16, 4},
		"w40": {17, 4},
		"w41": {18, 4},
		"w42": {19, 4},
		"w53": {20, 5},
		"w52": {21, 5},
		"w51": {22, 5},
		"j31": {23, 5},
		"w62": {24, 6},
		"w61": {25, 6},
		"w63": {26, 6},
		"h23": {27, 6},
		"w71": {28, 7},
	}

	for e, et := range expectedTimestamps {
		ev, err := h.Store.GetEvent(index[e])
		if err != nil {
			t.Fatal(err)
		}
		if r := ev.round; r == nil || *r != et.r {
			t.Fatalf("%s round should be %d, not %d", e, et.r, *r)
		}
		if ts := ev.lamportTimestamp; ts == nil || *ts != et.t {
			t.Fatalf("%s lamportTimestamp should be %d, not %d", e, et.t, *ts)
		}
	}

	/**************************************************************************/

	expectedWitnesses := map[int][]string{
		0: {"w00", "w01", "w02"},
		1: {"w10", "w11", "w12"},
		2: {"w20", "w21", "w22"},
		3: {"w30", "w31", "w32", "w33"},
		4: {"w40", "w41", "w42", "w43"},
		5: {"w51", "w52", "w53"},
		6: {"w61", "w62", "w63"},
		7: {"w71"},
	}

	for i := 0; i < 8; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(round.Witnesses()); l != len(expectedWitnesses[i]) {
			t.Fatalf("round %d should have %d witnesses, not %d", i, len(expectedWitnesses[i]), l)
		}
		for _, w := range expectedWitnesses[i] {
			if !contains(round.Witnesses(), index[w]) {
				t.Fatalf("round %d witnesses should contain %s", i, w)
			}
		}
	}
}

func TestR2DynDecideFame(t *testing.T) {
	h, index := initR2DynHashgraph(t)

	h.DivideRounds()
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}

	expectedEvents := map[int]map[string]roundEvent{
		0: {
			"w00": {Witness: true, Famous: common.True},
			"w01": {Witness: true, Famous: common.True},
			"w02": {Witness: true, Famous: common.True},
			"e10": {Witness: false, Famous: common.Undefined},
			"e21": {Witness: false, Famous: common.Undefined},
			"e12": {Witness: false, Famous: common.Undefined},
		},
		1: {
			"w10": {Witness: true, Famous: common.True},
			"w11": {Witness: true, Famous: common.True},
			"w12": {Witness: true, Famous: common.True},
			"f10": {Witness: false, Famous: common.Undefined},
		},
		2: {
			"w20": {Witness: true, Famous: common.True},
			"w21": {Witness: true, Famous: common.True},
			"w22": {Witness: true, Famous: common.True},
			"g21": {Witness: false, Famous: common.Undefined},
		},
		3: {
			"w30": {Witness: true, Famous: common.True},
			"w31": {Witness: true, Famous: common.True},
			"w32": {Witness: true, Famous: common.True},
			"w33": {Witness: true, Famous: common.True},
		},
		4: {
			"w40": {Witness: true, Famous: common.True},
			"w41": {Witness: true, Famous: common.True},
			"w42": {Witness: true, Famous: common.True},
			"w43": {Witness: true, Famous: common.True},
		},
		5: {
			"w51": {Witness: true, Famous: common.True},
			"w52": {Witness: true, Famous: common.True},
			"w53": {Witness: true, Famous: common.True},
			"j31": {Witness: false, Famous: common.Undefined},
		},
		6: {
			"w61": {Witness: true, Famous: common.Undefined},
			"w62": {Witness: true, Famous: common.Undefined},
			"w63": {Witness: true, Famous: common.Undefined},
			"h23": {Witness: false, Famous: common.Undefined},
		},
		7: {
			//created
			"w71": {Witness: true, Famous: common.Undefined},
		},
	}

	for i := 0; i < 8; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(round.CreatedEvents); l != len(expectedEvents[i]) {
			t.Fatalf("Round[%d].CreatedEvents should contain %d items, not %d", i, len(expectedEvents[i]), l)
		}
		for w, re := range expectedEvents[i] {
			if f := round.CreatedEvents[index[w]]; !reflect.DeepEqual(f, re) {
				t.Fatalf("%s should be %v; got %v", w, re, f)
			}
		}
	}
}

func TestR2DynDecideRoundReceived(t *testing.T) {
	h, index := initR2DynHashgraph(t)

	h.DivideRounds()
	h.DecideFame()
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}

	expectedConsensusEvents := map[int][]string{
		0: {},
		1: {index["w00"], index["w01"], index["w02"], index["e10"], index["e21"], index["e12"]},
		2: {index["w11"], index["w12"], index["w10"], index["f10"]},
		3: {index["w22"], index["w20"], index["w21"], index["g21"]},
		4: {index["w33"], index["w30"], index["w31"], index["w32"]},
		5: {index["w43"], index["w40"], index["w41"], index["w42"]},
		6: {},
		7: {},
	}

	for i := 0; i < 8; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(round.ReceivedEvents, expectedConsensusEvents[i]) {
			t.Fatalf("Round[%d].ReceivedEvents should be %v, %v", i, expectedConsensusEvents[i], round.ReceivedEvents)
		}
	}
}

func TestR2DynProcessDecidedRounds(t *testing.T) {
	h, index := initR2DynHashgraph(t)

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	if err := h.ProcessDecidedRounds(); err != nil {
		t.Fatal(err)
	}

	//--------------------------------------------------------------------------
	consensusEvents := h.Store.ConsensusEvents()

	for i, e := range consensusEvents {
		t.Logf("consensus[%d]: %s\n", i, getName(index, e))
	}

	if l := len(consensusEvents); l != 22 {
		t.Fatalf("length of consensus should be 22 not %d", l)
	}

	if ple := h.PendingLoadedEvents; ple != 9 {
		t.Fatalf("PendingLoadedEvents should be 9, not %d", ple)
	}

	//--------------------------------------------------------------------------

	for i := 0; i < 4; i++ {
		rr := i + 1

		frame, err := h.Store.GetFrame(rr)
		if err != nil {
			t.Fatal(err)
		}
		frameHash, _ := frame.Hash()

		ps, err := h.Store.GetPeerSet(rr)
		if err != nil {
			t.Fatal(err)
		}
		peersHash, _ := ps.Hash()

		block, err := h.Store.GetBlock(i)
		if err != nil {
			t.Fatal(err)
		}

		if brr := block.RoundReceived(); brr != rr {
			t.Fatalf("Block[%d].RoundReceived should be %d, not %d", i, rr, brr)
		}

		if bfh := block.FrameHash(); !reflect.DeepEqual(bfh, frameHash) {
			t.Fatalf("Block[%d].FrameHash should be %v, not %v", i, frameHash, bfh)
		}

		if bph := block.PeersHash(); !reflect.DeepEqual(bph, peersHash) {
			t.Fatalf("Block[%d].PeersHash should be %v, not %v", i, peersHash, bph)
		}
	}
}

/*
We insert Events into rounds to which the Event creator does not belong. These
Events should not be counted as witnesses and should not mess up the consensus.


			|   w41   |    |
		 -----/------------------
Round 3    h03   |    |    |
			|  \ |    |    |
			|    |  \ |    |
	        |    |    | \  |
		    |    |    |   x32 should not be a witness and should not count in strongly seen computations
			|    |    | /  |
			|    |   w32   R3
			|    |  / |
			|   w31   |
			| /  |    |
		   w30   |    |
			|  \ |    |
		 ------------------
Round 2		|    | \  |
			|    |   g21
			|    | /  |
			|   w21   |
			| /  |    |
		   w20   |    |
		    |  \ |    |
		    |    | \  |
		    |    |   w22
		 -----------/------
Round 1		|   f10   |
			| /  |    |
		   w10   |    |
		    |  \ |    |
		    |    | \  |
		    |    |   w12
		    |    |  / |
		    |   w11   |
		 -----/------------
Round 0	   e12   |    |
		    |  \ |    |
		    |    | \  |
		    |    |   e21
		    |    | /  |
		    |   e10   |
		    |  / |    |
		   w00  w01  w02
			|    |    |
		    R0   R1   R2
*/

func initUsurperHashgraph(t testing.TB) (*Hashgraph, map[string]string) {
	nodes, index, orderedEvents, peerSet := initHashgraphNodes(3)

	for i := range peerSet.Peers {
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, nil, []string{"", ""}, nodes[i].PubBytes, 0)
		nodes[i].signAndAddEvent(event, name, index, orderedEvents)
	}

	plays := []play{
		{1, 1, "w01", "w00", "e10", [][]byte{[]byte("e10")}, nil},
		{2, 1, "w02", "e10", "e21", [][]byte{[]byte("e21")}, nil},
		{0, 1, "w00", "e21", "e12", [][]byte{[]byte("e12")}, nil},
		{1, 2, "e10", "e12", "w11", [][]byte{[]byte("w11")}, nil},
		{2, 2, "e21", "w11", "w12", [][]byte{[]byte("w12")}, nil},
		{0, 2, "e12", "w12", "w10", [][]byte{[]byte("w10")}, nil},
		{1, 3, "w11", "w10", "f10", [][]byte{[]byte("f10")}, nil},
		{2, 3, "w12", "f10", "w22", [][]byte{[]byte("w22")}, nil},
		{0, 3, "w10", "w22", "w20", [][]byte{[]byte("w20")}, nil},
		{1, 4, "f10", "w20", "w21", [][]byte{[]byte("w21")}, nil},
		{2, 4, "w22", "w21", "g21", [][]byte{[]byte("g21")}, nil},
	}

	playEvents(plays, nodes, index, orderedEvents)

	hg := createHashgraph(false, orderedEvents, peerSet, t)

	/***************************************************************************
		Add Participant 3 (the usurper); new Peerset for Round10
		(far enough in the future)
	***************************************************************************/

	//create new node
	key3, _ := keys.GenerateECDSAKey()
	usurperNode := NewTestNode(key3)
	nodes = append(nodes, usurperNode)
	usurperPeer := peers.NewPeer(usurperNode.PubHex, "", "")
	index["R3"] = ""
	newPeerSet := peerSet.WithNewPeer(usurperPeer)

	//Set Round 10 PeerSet
	err := hg.Store.SetPeerSet(10, newPeerSet)
	if err != nil {
		t.Fatal(err)
	}

	plays = []play{
		{0, 4, "w20", "g21", "w30", [][]byte{[]byte("w30")}, nil},
		{1, 5, "w21", "w30", "w31", [][]byte{[]byte("w31")}, nil},
		{2, 5, "g21", "w31", "w32", [][]byte{[]byte("w32")}, nil},
		{3, 0, "R3", "w32", "x32", [][]byte{[]byte("x32")}, nil},
		{0, 5, "w30", "x32", "h03", [][]byte{[]byte("h03")}, nil},
		{1, 6, "w31", "h03", "w41", [][]byte{[]byte("w41")}, nil},
	}

	orderedEvents = &[]*Event{}

	playEvents(plays, nodes, index, orderedEvents)

	for i, ev := range *orderedEvents {
		if err := hg.InsertEvent(ev, true); err != nil {
			t.Fatalf("ERROR inserting event %d: %s\n", i, err)
		}
	}

	return hg, index
}

func TestUsurperDivideRounds(t *testing.T) {
	h, index := initUsurperHashgraph(t)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}

	/**************************************************************************/

	//[event] => {lamportTimestamp, round}
	type tr struct {
		t, r int
	}
	expectedTimestamps := map[string]tr{
		"w00": {0, 0},
		"w01": {0, 0},
		"w02": {0, 0},
		"e10": {1, 0},
		"e21": {2, 0},
		"e12": {3, 0},
		"w11": {4, 1},
		"w12": {5, 1},
		"w10": {6, 1},
		"f10": {7, 1},
		"w22": {8, 2},
		"w20": {9, 2},
		"w21": {10, 2},
		"g21": {11, 2},
		"w30": {12, 3},
		"w31": {13, 3},
		"w32": {14, 3},
		"x32": {15, 3},
		"h03": {16, 3},
		"w41": {17, 4},
	}

	for e, et := range expectedTimestamps {
		ev, err := h.Store.GetEvent(index[e])
		if err != nil {
			t.Fatal(err)
		}
		if r := ev.round; r == nil || *r != et.r {
			t.Fatalf("%s round should be %d, not %d", e, et.r, *r)
		}
		if ts := ev.lamportTimestamp; ts == nil || *ts != et.t {
			t.Fatalf("%s lamportTimestamp should be %d, not %d", e, et.t, *ts)
		}
	}

	/**************************************************************************/

	expectedWitnesses := map[int][]string{
		0: {"w00", "w01", "w02"},
		1: {"w10", "w11", "w12"},
		2: {"w20", "w21", "w22"},
		3: {"w30", "w31", "w32"},
		4: {"w41"},
	}

	for i := 0; i < 3; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(round.Witnesses()); l != len(expectedWitnesses[i]) {
			t.Fatalf("round %d should have %d witnesses, not %d", i, len(expectedWitnesses[i]), l)
		}
		for _, w := range expectedWitnesses[i] {
			if !contains(round.Witnesses(), index[w]) {
				t.Fatalf("round %d witnesses should contain %s", i, w)
			}
		}
	}
}

/*
	w80
	|
	w70
	|
	w60
    |
	w50
	|
	w40
	|
	w30
    |
	w20
	|
	w10
	|
    w00
	|
	R0
*/
func initMonologueHashgraph(t testing.TB) (*Hashgraph, map[string]string) {
	nodes, index, orderedEvents, peerSet := initHashgraphNodes(1)

	for i := range peerSet.Peers {
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, nil, []string{"", ""}, nodes[i].PubBytes, 0)
		nodes[i].signAndAddEvent(event, name, index, orderedEvents)
	}

	plays := []play{
		{0, 1, "w00", "", "w10", [][]byte{[]byte("w10")}, nil},
		{0, 2, "w10", "", "w20", [][]byte{[]byte("w20")}, nil},
		{0, 3, "w20", "", "w30", [][]byte{[]byte("w30")}, nil},
		{0, 4, "w30", "", "w40", [][]byte{[]byte("w40")}, nil},
		{0, 5, "w40", "", "w50", [][]byte{[]byte("w40")}, nil},
		{0, 6, "w50", "", "w60", [][]byte{[]byte("w60")}, nil},
		{0, 7, "w60", "", "w70", [][]byte{[]byte("w70")}, nil},
		{0, 8, "w70", "", "w80", [][]byte{[]byte("w80")}, nil},
	}

	playEvents(plays, nodes, index, orderedEvents)

	hg := createHashgraph(false, orderedEvents, peerSet, t)

	return hg, index
}

func TestMonologueDivideRounds(t *testing.T) {
	h, index := initMonologueHashgraph(t)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}

	/**************************************************************************/

	//[event] => {lamportTimestamp, round}
	type tr struct {
		t, r int
	}
	expectedTimestamps := map[string]tr{
		"w00": {0, 0},
		"w10": {1, 1},
		"w20": {2, 2},
		"w30": {3, 3},
		"w40": {4, 4},
		"w50": {5, 5},
		"w60": {6, 6},
		"w70": {7, 7},
		"w80": {8, 8},
	}

	for e, et := range expectedTimestamps {
		ev, err := h.Store.GetEvent(index[e])
		if err != nil {
			t.Fatal(err)
		}
		if r := ev.round; r == nil || *r != et.r {
			t.Fatalf("%s round should be %d, not %d", e, et.r, *r)
		}
		if ts := ev.lamportTimestamp; ts == nil || *ts != et.t {
			t.Fatalf("%s lamportTimestamp should be %d, not %d", e, et.t, *ts)
		}
	}

	/**************************************************************************/

	expectedWitnesses := map[int][]string{
		0: {"w00"},
		1: {"w10"},
		2: {"w20"},
		3: {"w30"},
		4: {"w40"},
		5: {"w50"},
		6: {"w60"},
		7: {"w70"},
		8: {"w80"},
	}

	for i := 0; i < 9; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(round.Witnesses()); l != len(expectedWitnesses[i]) {
			t.Fatalf("round %d should have %d witnesses, not %d", i, len(expectedWitnesses[i]), l)
		}
		for _, w := range expectedWitnesses[i] {
			if !contains(round.Witnesses(), index[w]) {
				t.Fatalf("round %d witnesses should contain %s", i, w)
			}
		}
	}
}

func TestMonologueDecideFame(t *testing.T) {
	h, index := initMonologueHashgraph(t)

	h.DivideRounds()
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}

	expectedEvents := map[int]map[string]roundEvent{
		0: {
			"w00": {Witness: true, Famous: common.True},
		},
		1: {
			"w10": {Witness: true, Famous: common.True},
		},
		2: {
			"w20": {Witness: true, Famous: common.True},
		},
		3: {
			"w30": {Witness: true, Famous: common.True},
		},
		4: {
			"w40": {Witness: true, Famous: common.True},
		},
		5: {
			"w50": {Witness: true, Famous: common.True},
		},
		6: {
			"w60": {Witness: true, Famous: common.True},
		},
		7: {
			"w70": {Witness: true, Famous: common.Undefined},
		},
		8: {
			"w80": {Witness: true, Famous: common.Undefined},
		},
	}

	for i := 0; i < 9; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(round.CreatedEvents); l != len(expectedEvents[i]) {
			t.Fatalf("Round[%d].CreatedEvents should contain %d items, not %d", i, len(expectedEvents[i]), l)
		}
		for w, re := range expectedEvents[i] {
			if f := round.CreatedEvents[index[w]]; !reflect.DeepEqual(f, re) {
				t.Fatalf("%s should be %v; got %v", w, re, f)
			}
		}
	}
}

func TestMonologueDecideRoundReceived(t *testing.T) {
	h, index := initMonologueHashgraph(t)

	h.DivideRounds()
	h.DecideFame()
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}

	expectedConsensusEvents := map[int][]string{
		0: {},
		1: {index["w00"]},
		2: {index["w10"]},
		3: {index["w20"]},
		4: {index["w30"]},
		5: {index["w40"]},
		6: {index["w50"]},
	}

	for i := 0; i < 7; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(round.ReceivedEvents, expectedConsensusEvents[i]) {
			t.Fatalf("Round[%d].ReceivedEvents should be %v, %v", i, expectedConsensusEvents[i], round.ReceivedEvents)
		}
	}
}

// func TestR2DynProcessDecidedRounds(t *testing.T) {
// 	h, index := initR2DynHashgraph(t)

// 	h.DivideRounds()
// 	h.DecideFame()
// 	h.DecideRoundReceived()
// 	if err := h.ProcessDecidedRounds(); err != nil {
// 		t.Fatal(err)
// 	}

// 	//--------------------------------------------------------------------------
// 	consensusEvents := h.Store.ConsensusEvents()

// 	for i, e := range consensusEvents {
// 		t.Logf("consensus[%d]: %s\n", i, getName(index, e))
// 	}

// 	if l := len(consensusEvents); l != 22 {
// 		t.Fatalf("length of consensus should be 22 not %d", l)
// 	}

// 	if ple := h.PendingLoadedEvents; ple != 9 {
// 		t.Fatalf("PendingLoadedEvents should be 9, not %d", ple)
// 	}

// 	//--------------------------------------------------------------------------

// 	for i := 0; i < 4; i++ {
// 		rr := i + 1

// 		frame, err := h.Store.GetFrame(rr)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		frameHash, _ := frame.Hash()

// 		ps, err := h.Store.GetPeerSet(rr)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		peersHash, _ := ps.Hash()

// 		block, err := h.Store.GetBlock(i)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		if brr := block.RoundReceived(); brr != rr {
// 			t.Fatalf("Block[%d].RoundReceived should be %d, not %d", i, rr, brr)
// 		}

// 		if bfh := block.FrameHash(); !reflect.DeepEqual(bfh, frameHash) {
// 			t.Fatalf("Block[%d].FrameHash should be %v, not %v", i, frameHash, bfh)
// 		}

// 		if bph := block.PeersHash(); !reflect.DeepEqual(bph, peersHash) {
// 			t.Fatalf("Block[%d].PeersHash should be %v, not %v", i, peersHash, bph)
// 		}
// 	}
// }
