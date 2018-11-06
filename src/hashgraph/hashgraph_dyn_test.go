package hashgraph

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*
We introduce a new participant at Round 2.

Round 4
P:[0,1,2,3] |    |    |    |
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

	for i, peer := range peerSet.Peers {
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, []string{rootSelfParent(peer.ID), ""}, nodes[i].Pub, 0)
		nodes[i].signAndAddEvent(event, name, index, orderedEvents)
	}

	plays := []play{
		play{1, 1, "w01", "w00", "e10", [][]byte{[]byte("e10")}, nil},
		play{2, 1, "w02", "e10", "e21", [][]byte{[]byte("e21")}, nil},
		play{0, 1, "w00", "e21", "e12", [][]byte{[]byte("e12")}, nil},
		play{1, 2, "e10", "e12", "w11", [][]byte{[]byte("w11")}, nil},
		play{2, 2, "e21", "w11", "w12", [][]byte{[]byte("w12")}, nil},
		play{0, 2, "e12", "w12", "w10", [][]byte{[]byte("w10")}, nil},
		play{1, 3, "w11", "w10", "f10", [][]byte{[]byte("f10")}, nil},
		play{2, 3, "w12", "f10", "w22", [][]byte{[]byte("w22")}, nil},
		play{0, 3, "w10", "w22", "w20", [][]byte{[]byte("w20")}, nil},
		play{1, 4, "f10", "w20", "w21", [][]byte{[]byte("w21")}, nil},
		play{2, 4, "w22", "w21", "g21", [][]byte{[]byte("g21")}, nil},
	}

	playEvents(plays, nodes, index, orderedEvents)

	hg := createHashgraph(false, orderedEvents, peerSet, common.NewTestLogger(t).WithField("test", "R2D"))

	/***************************************************************************
		Add Participant 3; new Peerset for Round2
	***************************************************************************/

	//create new node
	key3, _ := crypto.GenerateECDSAKey()
	node3 := NewTestNode(key3)
	nodes = append(nodes, node3)
	peer3 := peers.NewPeer(node3.PubHex, "")
	index["R3"] = rootSelfParent(peer3.ID)
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
		play{3, 0, "R3", "g21", "w33", [][]byte{[]byte("w33")}, nil},
		play{0, 4, "w20", "w33", "w30", [][]byte{[]byte("w30")}, nil},
		play{1, 5, "w21", "w30", "w31", [][]byte{[]byte("w31")}, nil},
		play{2, 5, "g21", "w31", "w32", [][]byte{[]byte("w32")}, nil},
		play{3, 1, "w33", "w32", "w43", [][]byte{[]byte("w43")}, nil},
		play{0, 5, "w30", "w43", "w40", [][]byte{[]byte("w40")}, nil},
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
		"w00": tr{0, 0},
		"w01": tr{0, 0},
		"w02": tr{0, 0},
		"e10": tr{1, 0},
		"e21": tr{2, 0},
		"e12": tr{3, 0},
		"w11": tr{4, 1},
		"w12": tr{5, 1},
		"w10": tr{6, 1},
		"f10": tr{7, 1},
		"w22": tr{8, 2},
		"w20": tr{9, 2},
		"w21": tr{10, 2},
		"g21": tr{11, 2},
		"w33": tr{12, 3},
		"w30": tr{13, 3},
		"w31": tr{14, 3},
		"w32": tr{15, 3},
		"w43": tr{16, 4},
		"w40": tr{17, 4},
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

	round0, err := h.Store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(round0.Witnesses()); l != 3 {
		t.Fatalf("round 0 should have 3 witnesses, not %d", l)
	}
	if !contains(round0.Witnesses(), index["w00"]) {
		t.Fatalf("round 0 witnesses should contain w00")
	}
	if !contains(round0.Witnesses(), index["w01"]) {
		t.Fatalf("round 0 witnesses should contain w01")
	}
	if !contains(round0.Witnesses(), index["w02"]) {
		t.Fatalf("round 0 witnesses should contain w02")
	}

	round2, err := h.Store.GetRound(2)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(round2.Witnesses()); l != 3 {
		t.Fatalf("round 2 should have 3 witnesses, not %d", l)
	}
	if !contains(round2.Witnesses(), index["w22"]) {
		t.Fatalf("round 2 witnesses should contain w22")
	}
	if !contains(round2.Witnesses(), index["w20"]) {
		t.Fatalf("round 2 witnesses should contain w20")
	}
	if !contains(round2.Witnesses(), index["w21"]) {
		t.Fatalf("round 2 witnesses should contain w21")
	}

	round3, err := h.Store.GetRound(3)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(round3.Witnesses()); l != 4 {
		t.Fatalf("round 3 should have 4 witnesses, not %d", l)
	}
	if !contains(round3.Witnesses(), index["w33"]) {
		t.Fatalf("round 3 witnesses should contain w33")
	}
	if !contains(round3.Witnesses(), index["w30"]) {
		t.Fatalf("round 3 witnesses should contain w30")
	}
	if !contains(round3.Witnesses(), index["w31"]) {
		t.Fatalf("round 3 witnesses should contain w31")
	}
	if !contains(round3.Witnesses(), index["w32"]) {
		t.Fatalf("round 3 witnesses should contain w32")
	}

	round4, err := h.Store.GetRound(4)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(round4.Witnesses()); l != 2 {
		t.Fatalf("round 4 should have 2 witnesses, not %d", l)
	}
	if !contains(round4.Witnesses(), index["w43"]) {
		t.Fatalf("round 4 witnesses should contain w43")
	}
	if !contains(round4.Witnesses(), index["w40"]) {
		t.Fatalf("round 4 witnesses should contain w40")
	}

	/**************************************************************************/

	firstPeerSet, err := h.Store.GetPeerSet(0)
	if !reflect.DeepEqual(round0.PeerSet, firstPeerSet) {
		t.Fatalf("Round0 PeerSet should be %v, not %v", firstPeerSet, round0.PeerSet)
	}

	lastPeerSet, err := h.Store.GetLastPeerSet()
	if !reflect.DeepEqual(round2.PeerSet, lastPeerSet) {
		t.Fatalf("Round2 PeerSet should be %v, not %v", lastPeerSet, round2.PeerSet)
	}
	if !reflect.DeepEqual(round3.PeerSet, lastPeerSet) {
		t.Fatalf("Round3 PeerSet should be %v, not %v", lastPeerSet, round3.PeerSet)
	}
}

func TestR2DynDecideFame(t *testing.T) {
	h, index := initR2DynHashgraph(t)

	h.DivideRounds()
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}

	round0, err := h.Store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}
	if f := round0.Events[index["w00"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w00 should be famous; got %v", f)
	}
	if f := round0.Events[index["w01"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w01 should be famous; got %v", f)
	}
	if f := round0.Events[index["w02"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w02 should be famous; got %v", f)
	}

	round1, err := h.Store.GetRound(1)
	if err != nil {
		t.Fatal(err)
	}
	if f := round1.Events[index["w11"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w11 should be famous; got %v", f)
	}
	if f := round1.Events[index["w12"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w12 should be famous; got %v", f)
	}
	if f := round1.Events[index["w10"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w10 should be famous; got %v", f)
	}

	round2, err := h.Store.GetRound(2)
	if err != nil {
		t.Fatal(err)
	}
	if f := round2.Events[index["w22"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w22 should be famous; got %v", f)
	}
	if f := round2.Events[index["w20"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w20 should be famous; got %v", f)
	}
	if f := round2.Events[index["w21"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("w21 should be famous; got %v", f)
	}

	round3, err := h.Store.GetRound(3)
	if err != nil {
		t.Fatal(err)
	}
	if f := round3.Events[index["w33"]]; !(f.Witness && f.Famous == Undefined) {
		t.Fatalf("w33 should not be Undefined; got %v", f)
	}
	if f := round3.Events[index["w30"]]; !(f.Witness && f.Famous == Undefined) {
		t.Fatalf("w30 should not be Undefined; got %v", f)
	}
	if f := round3.Events[index["w31"]]; !(f.Witness && f.Famous == Undefined) {
		t.Fatalf("w31 should not be Undefined; got %v", f)
	}
	if f := round3.Events[index["w32"]]; !(f.Witness && f.Famous == Undefined) {
		t.Fatalf("w32 should not be Undefined; got %v", f)
	}

}

func TestR2DynDecideRoundReceived(t *testing.T) {
	h, index := initR2DynHashgraph(t)

	h.DivideRounds()
	h.DecideFame()
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}

	round0, err := h.Store.GetRound(0)
	if err != nil {
		t.Fatalf("Could not retrieve Round 0. %s", err)
	}
	if ce := len(round0.ConsensusEvents()); ce != 0 {
		t.Fatalf("Round 0 should contain 0 ConsensusEvents, not %d", ce)
	}

	round1, err := h.Store.GetRound(1)
	if err != nil {
		t.Fatalf("Could not retrieve Round 1. %s", err)
	}
	if ce := len(round1.ConsensusEvents()); ce != 6 {
		t.Fatalf("Round 1 should contain 6 ConsensusEvents, not %d", ce)
	}

	round2, err := h.Store.GetRound(2)
	if err != nil {
		t.Fatalf("Could not retrieve Round 2. %s", err)
	}
	if ce := len(round2.ConsensusEvents()); ce != 4 {
		t.Fatalf("Round 2 should contain 4 ConsensusEvents, not %d", ce)
	}

	round3, err := h.Store.GetRound(3)
	if err != nil {
		t.Fatalf("Could not retrieve Round 3. %s", err)
	}
	if ce := len(round3.ConsensusEvents()); ce != 0 {
		t.Fatalf("Round 3 should contain 0 ConsensusEvents, not %d", ce)
	}

	expectedUndeterminedEvents := []string{
		index["w22"],
		index["w20"],
		index["w21"],
		index["g21"],
		index["w33"],
		index["w30"],
		index["w31"],
		index["w32"],
		index["w43"],
		index["w40"],
	}

	for i, eue := range expectedUndeterminedEvents {
		if ue := h.UndeterminedEvents[i]; ue != eue {
			t.Fatalf("UndeterminedEvents[%d] should be %s, not %s", i, eue, ue)
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

	if l := len(consensusEvents); l != 10 {
		t.Fatalf("length of consensus should be 10 not %d", l)
	}

	if ple := h.PendingLoadedEvents; ple != 10 {
		t.Fatalf("PendingLoadedEvents should be 10, not %d", ple)
	}

	//Block 0 ------------------------------------------------------------------
	block0, err := h.Store.GetBlock(0)
	if err != nil {
		t.Fatalf("Store should contain a block with Index 0: %v", err)
	}

	if ind := block0.Index(); ind != 0 {
		t.Fatalf("Block0's Index should be 0, not %d", ind)
	}

	if rr := block0.RoundReceived(); rr != 1 {
		t.Fatalf("Block0's RoundReceived should be 1, not %d", rr)
	}

	if l := len(block0.Transactions()); l != 6 {
		t.Fatalf("Block0 should contain 6 transactions, not %d", l)
	}

	frame1, err := h.GetFrame(1)
	frame1Hash, err := frame1.Hash()
	if !reflect.DeepEqual(block0.FrameHash(), frame1Hash) {
		t.Fatalf("Block0.FrameHash should be %v, not %v", frame1Hash, block0.FrameHash())
	}

	firstPeerSet, err := h.Store.GetPeerSet(0)
	if !reflect.DeepEqual(frame1.Peers, firstPeerSet.Peers) {
		t.Fatalf("Frame1 Peers should be %v, not %v", frame1.Peers, firstPeerSet.Peers)
	}

	//Block 1 ------------------------------------------------------------------
	block1, err := h.Store.GetBlock(1)
	if err != nil {
		t.Fatalf("Store should contain a block with Index 1: %v", err)
	}

	if ind := block1.Index(); ind != 1 {
		t.Fatalf("Block1's Index should be 1, not %d", ind)
	}

	if rr := block1.RoundReceived(); rr != 2 {
		t.Fatalf("Block1's RoundReceived should be 2, not %d", rr)
	}

	if l := len(block1.Transactions()); l != 4 {
		t.Fatalf("Block1 should contain 4 transactions, not %d", l)
	}

	frame2, err := h.GetFrame(2)
	frame2Hash, err := frame2.Hash()
	if !reflect.DeepEqual(block1.FrameHash(), frame2Hash) {
		t.Fatalf("Block1.FrameHash should be %v, not %v", frame2Hash, block1.FrameHash())
	}

	lastPeerSet, err := h.Store.GetLastPeerSet()
	if !reflect.DeepEqual(frame2.Peers, lastPeerSet.Peers) {
		t.Fatalf("Frame2 Peers should be %v, not %v", frame2.Peers, lastPeerSet.Peers)
	}
}
