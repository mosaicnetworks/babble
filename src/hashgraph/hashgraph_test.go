package hashgraph

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

var (
	cacheSize = 100
	n         = 3
	badgerDir = "test_data/badger"
)

type TestNode struct {
	ID     int
	Pub    []byte
	PubHex string
	Key    *ecdsa.PrivateKey
	Events []*Event
}

func NewTestNode(key *ecdsa.PrivateKey) TestNode {
	pub := crypto.FromECDSAPub(&key.PublicKey)
	ID := common.Hash32(pub)
	node := TestNode{
		ID:     ID,
		Key:    key,
		Pub:    pub,
		PubHex: fmt.Sprintf("0x%X", pub),
		Events: []*Event{},
	}
	return node
}

func (node *TestNode) signAndAddEvent(event *Event, name string, index map[string]string, orderedEvents *[]*Event) {
	event.Sign(node.Key)
	node.Events = append(node.Events, event)
	index[name] = event.Hex()
	*orderedEvents = append(*orderedEvents, event)
}

type ancestryItem struct {
	descendant, ancestor string
	val                  bool
	err                  bool
}

type roundItem struct {
	event string
	round int
}

type play struct {
	to          int
	index       int
	selfParent  string
	otherParent string
	name        string
	txPayload   [][]byte
	sigPayload  []BlockSignature
}

func testLogger(t testing.TB) *logrus.Entry {
	return common.NewTestLogger(t).WithField("id", "test")
}

/* Initialisation functions */

func initHashgraphNodes(n int) ([]TestNode, map[string]string, *[]*Event, *peers.PeerSet) {
	index := make(map[string]string)
	nodes := []TestNode{}
	orderedEvents := &[]*Event{}
	keys := map[string]*ecdsa.PrivateKey{}
	pirs := []*peers.Peer{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pub := crypto.FromECDSAPub(&key.PublicKey)
		pubHex := fmt.Sprintf("0x%X", pub)
		pirs = append(pirs, peers.NewPeer(pubHex, ""))
		keys[pubHex] = key
		nodes = append(nodes, NewTestNode(key))
	}

	peerSet := peers.NewPeerSet(pirs)

	return nodes, index, orderedEvents, peerSet
}

func playEvents(plays []play, nodes []TestNode, index map[string]string, orderedEvents *[]*Event) {
	for _, p := range plays {
		e := NewEvent(p.txPayload,
			nil,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		nodes[p.to].signAndAddEvent(e, p.name, index, orderedEvents)
	}
}

func createHashgraph(db bool, orderedEvents *[]*Event, peerSet *peers.PeerSet, logger *logrus.Entry) *Hashgraph {
	var store Store
	if db {
		var err error
		store, err = NewBadgerStore(peerSet, cacheSize, badgerDir)
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		store = NewInmemStore(peerSet, cacheSize)
	}

	hashgraph := NewHashgraph(peerSet, store, DummyInternalCommitCallback, logger)

	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev, true); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}

	return hashgraph
}

func initHashgraphFull(plays []play, db bool, n int, logger *logrus.Entry) (*Hashgraph, map[string]string, *[]*Event) {
	nodes, index, orderedEvents, peerSet := initHashgraphNodes(n)

	for i, n := range nodes {
		event := NewEvent(nil, nil, nil, []string{rootSelfParent(n.ID), ""}, n.Pub, 0)
		n.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
	}

	playEvents(plays, nodes, index, orderedEvents)

	hashgraph := createHashgraph(db, orderedEvents, peerSet, logger)

	return hashgraph, index, orderedEvents
}

/*  */

/*
|  e12  |
|   | \ |
|  s10 e20
|   | / |
|   /   |
| / |   |
s00 |  s20
|   |   |
e01 |   |
| \ |   |
e0  e1  e2
0   1   2
*/
func initHashgraph(t *testing.T) (*Hashgraph, map[string]string) {
	plays := []play{
		play{0, 1, "e0", "e1", "e01", nil, nil},
		play{2, 1, "e2", "", "s20", nil, nil},
		play{1, 1, "e1", "", "s10", nil, nil},
		play{0, 2, "e01", "", "s00", nil, nil},
		play{2, 2, "s20", "s00", "e20", nil, nil},
		play{1, 2, "s10", "e20", "e12", nil, nil},
	}

	h, index, orderedEvents := initHashgraphFull(plays, false, n, testLogger(t))

	for i, ev := range *orderedEvents {
		if err := h.initEventCoordinates(ev); err != nil {
			t.Fatalf("%d: %s", i, err)
		}

		if err := h.Store.SetEvent(ev); err != nil {
			t.Fatalf("%d: %s", i, err)
		}

		if err := h.updateAncestorFirstDescendant(ev); err != nil {
			t.Fatalf("%d: %s", i, err)
		}
	}

	return h, index
}

func TestAncestor(t *testing.T) {
	h, index := initHashgraph(t)

	expected := []ancestryItem{
		//first generation
		ancestryItem{"e01", "e0", true, false},
		ancestryItem{"e01", "e1", true, false},
		ancestryItem{"s00", "e01", true, false},
		ancestryItem{"s20", "e2", true, false},
		ancestryItem{"e20", "s00", true, false},
		ancestryItem{"e20", "s20", true, false},
		ancestryItem{"e12", "e20", true, false},
		ancestryItem{"e12", "s10", true, false},
		//second generation
		ancestryItem{"s00", "e0", true, false},
		ancestryItem{"s00", "e1", true, false},
		ancestryItem{"e20", "e01", true, false},
		ancestryItem{"e20", "e2", true, false},
		ancestryItem{"e12", "e1", true, false},
		ancestryItem{"e12", "s20", true, false},
		//third generation
		ancestryItem{"e20", "e0", true, false},
		ancestryItem{"e20", "e1", true, false},
		ancestryItem{"e20", "e2", true, false},
		ancestryItem{"e12", "e01", true, false},
		ancestryItem{"e12", "e0", true, false},
		ancestryItem{"e12", "e1", true, false},
		ancestryItem{"e12", "e2", true, false},
		//false positive
		ancestryItem{"e01", "e2", false, false},
		ancestryItem{"s00", "e2", false, false},
		ancestryItem{"e0", "", false, true},
		ancestryItem{"s00", "", false, true},
		ancestryItem{"e12", "", false, true},
	}

	for _, exp := range expected {
		a, err := h.ancestor(index[exp.descendant], index[exp.ancestor])
		if err != nil && !exp.err {
			t.Fatalf("Error computing ancestor(%s, %s). Err: %v", exp.descendant, exp.ancestor, err)
		}
		if a != exp.val {
			t.Fatalf("ancestor(%s, %s) should be %v, not %v", exp.descendant, exp.ancestor, exp.val, a)
		}
	}
}

func TestSelfAncestor(t *testing.T) {
	h, index := initHashgraph(t)

	expected := []ancestryItem{
		//1 generation
		ancestryItem{"e01", "e0", true, false},
		ancestryItem{"s00", "e01", true, false},
		//1 generation false negative
		ancestryItem{"e01", "e1", false, false},
		ancestryItem{"e12", "e20", false, false},
		ancestryItem{"s20", "e1", false, false},
		ancestryItem{"s20", "", false, true},
		//2 generations
		ancestryItem{"e20", "e2", true, false},
		ancestryItem{"e12", "e1", true, false},
		//2 generations false negatives
		ancestryItem{"e20", "e0", false, false},
		ancestryItem{"e12", "e2", false, false},
		ancestryItem{"e20", "e01", false, false},
	}

	for _, exp := range expected {
		a, err := h.selfAncestor(index[exp.descendant], index[exp.ancestor])
		if err != nil && !exp.err {
			t.Fatalf("Error computing selfAncestor(%s, %s). Err: %v", exp.descendant, exp.ancestor, err)
		}
		if a != exp.val {
			t.Fatalf("selfAncestor(%s, %s) should be %v, not %v", exp.descendant, exp.ancestor, exp.val, a)
		}
	}
}

func TestSee(t *testing.T) {
	h, index := initHashgraph(t)

	expected := []ancestryItem{
		ancestryItem{"e01", "e0", true, false},
		ancestryItem{"e01", "e1", true, false},
		ancestryItem{"e20", "e0", true, false},
		ancestryItem{"e20", "e01", true, false},
		ancestryItem{"e12", "e01", true, false},
		ancestryItem{"e12", "e0", true, false},
		ancestryItem{"e12", "e1", true, false},
		ancestryItem{"e12", "s20", true, false},
	}

	for _, exp := range expected {
		a, err := h.see(index[exp.descendant], index[exp.ancestor])
		if err != nil && !exp.err {
			t.Fatalf("Error computing see(%s, %s). Err: %v", exp.descendant, exp.ancestor, err)
		}
		if a != exp.val {
			t.Fatalf("see(%s, %s) should be %v, not %v", exp.descendant, exp.ancestor, exp.val, a)
		}
	}
}

func TestLamportTimestamp(t *testing.T) {
	h, index := initHashgraph(t)

	expectedTimestamps := map[string]int{
		"e0":  0,
		"e1":  0,
		"e2":  0,
		"e01": 1,
		"s10": 1,
		"s20": 1,
		"s00": 2,
		"e20": 3,
		"e12": 4,
	}

	for e, ets := range expectedTimestamps {
		ts, err := h.lamportTimestamp(index[e])
		if err != nil {
			t.Fatalf("Error computing lamportTimestamp(%s). Err: %s", e, err)
		}
		if ts != ets {
			t.Fatalf("%s LamportTimestamp should be %d, not %d", e, ets, ts)
		}
	}
}

/*
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
func TestFork(t *testing.T) {
	index := make(map[string]string)
	nodes := []TestNode{}
	pirs := []*peers.Peer{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewTestNode(key)
		nodes = append(nodes, node)
		pirs = append(pirs, peers.NewPeer(node.PubHex, ""))
	}

	peerSet := peers.NewPeerSet(pirs)

	store := NewInmemStore(peerSet, cacheSize)
	hashgraph := NewHashgraph(peerSet, store, DummyInternalCommitCallback, testLogger(t))

	for i, node := range nodes {
		event := NewEvent(nil, nil, nil, []string{"", ""}, node.Pub, 0)
		event.Sign(node.Key)
		index[fmt.Sprintf("e%d", i)] = event.Hex()
		hashgraph.InsertEvent(event, true)
	}

	//a and e2 need to have different hashes
	eventA := NewEvent([][]byte{[]byte("yo")}, nil, nil, []string{"", ""}, nodes[2].Pub, 0)
	eventA.Sign(nodes[2].Key)
	index["a"] = eventA.Hex()
	if err := hashgraph.InsertEvent(eventA, true); err == nil {
		t.Fatal("InsertEvent should return error for 'a'")
	}

	event01 := NewEvent(nil, nil, nil,
		[]string{index["e0"], index["a"]}, //e0 and a
		nodes[0].Pub, 1)
	event01.Sign(nodes[0].Key)
	index["e01"] = event01.Hex()
	if err := hashgraph.InsertEvent(event01, true); err == nil {
		t.Fatal("InsertEvent should return error for e01")
	}

	event20 := NewEvent(nil, nil, nil,
		[]string{index["e2"], index["e01"]}, //e2 and e01
		nodes[2].Pub, 1)
	event20.Sign(nodes[2].Key)
	index["e20"] = event20.Hex()
	if err := hashgraph.InsertEvent(event20, true); err == nil {
		t.Fatal("InsertEvent should return error for e20")
	}
}

/*
|  s11  |
|   |   |
|   f1  |
|  /|   |
| / s10 |
|/  |   |
e02 |   |
| \ |   |
|   \   |
|   | \ |
s00 |  e21
|   | / |
|  e10  s20
| / |   |
e0  e1  e2
0   1    2
*/

func initRoundHashgraph(t *testing.T) (*Hashgraph, map[string]string) {
	plays := []play{
		play{1, 1, "e1", "e0", "e10", nil, nil},
		play{2, 1, "e2", "", "s20", nil, nil},
		play{0, 1, "e0", "", "s00", nil, nil},
		play{2, 2, "s20", "e10", "e21", nil, nil},
		play{0, 2, "s00", "e21", "e02", nil, nil},
		play{1, 2, "e10", "", "s10", nil, nil},
		play{1, 3, "s10", "e02", "f1", nil, nil},
		play{1, 4, "f1", "", "s11", [][]byte{[]byte("abc")}, nil},
	}

	h, index, _ := initHashgraphFull(plays, false, n, testLogger(t))

	lastPeerSet, err := h.Store.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}

	//Set Rounds manually; this would normally be handled by DivideRounds()
	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e1"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, &RoundInfo{CreatedEvents: round0Witnesses})
	h.Store.SetPeerSet(0, lastPeerSet)

	round1Witnesses := make(map[string]RoundEvent)
	round1Witnesses[index["f1"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(1, &RoundInfo{CreatedEvents: round1Witnesses})
	h.Store.SetPeerSet(1, lastPeerSet)

	return h, index
}

func TestInsertEvent(t *testing.T) {
	h, index := initRoundHashgraph(t)

	t.Run("Check Event Coordinates", func(t *testing.T) {

		lastPeerSet, err := h.Store.GetLastPeerSet()
		if err != nil {
			t.Fatal(err)
		}

		//e0
		e0, err := h.Store.GetEvent(index["e0"])
		if err != nil {
			t.Fatal(err)
		}

		if !(e0.Body.selfParentIndex == -1 &&
			e0.Body.otherParentCreatorID == -1 &&
			e0.Body.otherParentIndex == -1 &&
			e0.Body.creatorID == lastPeerSet.ByPubKey[e0.Creator()].ID) {
			t.Fatalf("Invalid wire info on e0")
		}

		expectedFirstDescendants := CoordinatesMap{
			lastPeerSet.PubKeys()[0]: EventCoordinates{index["e0"], 0},
			lastPeerSet.PubKeys()[1]: EventCoordinates{index["e10"], 1},
			lastPeerSet.PubKeys()[2]: EventCoordinates{index["e21"], 2},
		}

		expectedLastAncestors := CoordinatesMap{
			lastPeerSet.PubKeys()[0]: EventCoordinates{index["e0"], 0},
			lastPeerSet.PubKeys()[1]: EventCoordinates{"", -1},
			lastPeerSet.PubKeys()[2]: EventCoordinates{"", -1},
		}

		if !reflect.DeepEqual(e0.firstDescendants, expectedFirstDescendants) {
			t.Fatalf("e0 firstDescendants should be %v, not %v",
				expectedFirstDescendants,
				e0.firstDescendants)
		}
		if !reflect.DeepEqual(e0.lastAncestors, expectedLastAncestors) {
			t.Fatal("e0 lastAncestors not good")
		}

		//e21
		e21, err := h.Store.GetEvent(index["e21"])
		if err != nil {
			t.Fatal(err)
		}

		e10, err := h.Store.GetEvent(index["e10"])
		if err != nil {
			t.Fatal(err)
		}

		if !(e21.Body.selfParentIndex == 1 &&
			e21.Body.otherParentCreatorID == lastPeerSet.ByPubKey[e10.Creator()].ID &&
			e21.Body.otherParentIndex == 1 &&
			e21.Body.creatorID == lastPeerSet.ByPubKey[e21.Creator()].ID) {
			t.Fatalf("Invalid wire info on e21")
		}

		expectedFirstDescendants = CoordinatesMap{
			lastPeerSet.PubKeys()[0]: EventCoordinates{index["e02"], 2},
			lastPeerSet.PubKeys()[1]: EventCoordinates{index["f1"], 3},
			lastPeerSet.PubKeys()[2]: EventCoordinates{index["e21"], 2},
		}

		expectedLastAncestors = CoordinatesMap{
			lastPeerSet.PubKeys()[0]: EventCoordinates{index["e0"], 0},
			lastPeerSet.PubKeys()[1]: EventCoordinates{index["e10"], 1},
			lastPeerSet.PubKeys()[2]: EventCoordinates{index["e21"], 2},
		}

		if !reflect.DeepEqual(e21.firstDescendants, expectedFirstDescendants) {
			t.Fatal("e21 firstDescendants not good")
		}
		if !reflect.DeepEqual(e21.lastAncestors, expectedLastAncestors) {
			t.Fatal("e21 lastAncestors not good")
		}

		//f1
		f1, err := h.Store.GetEvent(index["f1"])
		if err != nil {
			t.Fatal(err)
		}

		if !(f1.Body.selfParentIndex == 2 &&
			f1.Body.otherParentCreatorID == lastPeerSet.ByPubKey[e0.Creator()].ID &&
			f1.Body.otherParentIndex == 2 &&
			f1.Body.creatorID == lastPeerSet.ByPubKey[f1.Creator()].ID) {
			t.Fatalf("Invalid wire info on f1")
		}

		expectedFirstDescendants = CoordinatesMap{
			lastPeerSet.PubKeys()[0]: EventCoordinates{"", math.MaxInt32},
			lastPeerSet.PubKeys()[1]: EventCoordinates{index["f1"], 3},
			lastPeerSet.PubKeys()[2]: EventCoordinates{"", math.MaxInt32},
		}

		expectedLastAncestors = CoordinatesMap{
			lastPeerSet.PubKeys()[0]: EventCoordinates{index["e02"], 2},
			lastPeerSet.PubKeys()[1]: EventCoordinates{index["f1"], 3},
			lastPeerSet.PubKeys()[2]: EventCoordinates{index["e21"], 2},
		}

		if !reflect.DeepEqual(f1.firstDescendants, expectedFirstDescendants) {
			t.Fatal("f1 firstDescendants not good")
		}
		if !reflect.DeepEqual(f1.lastAncestors, expectedLastAncestors) {
			t.Fatal("f1 lastAncestors not good")
		}
	})

	t.Run("Check UndeterminedEvents", func(t *testing.T) {

		expectedUndeterminedEvents := []string{
			index["e0"],
			index["e1"],
			index["e2"],
			index["e10"],
			index["s20"],
			index["s00"],
			index["e21"],
			index["e02"],
			index["s10"],
			index["f1"],
			index["s11"]}

		for i, eue := range expectedUndeterminedEvents {
			if ue := h.UndeterminedEvents[i]; ue != eue {
				t.Fatalf("UndeterminedEvents[%d] should be %s, not %s", i, eue, ue)
			}
		}

		//Pending loaded Events
		// 3 Events with index 0,
		// 1 Event with non-empty Transactions
		//= 4 Loaded Events
		if ple := h.PendingLoadedEvents; ple != 4 {
			t.Fatalf("PendingLoadedEvents should be 4, not %d", ple)
		}
	})
}

func TestReadWireInfo(t *testing.T) {
	h, index := initRoundHashgraph(t)

	for k, evh := range index {
		ev, err := h.Store.GetEvent(evh)
		if err != nil {
			t.Fatal(err)
		}

		evWire := ev.ToWire()

		evFromWire, err := h.ReadWireInfo(evWire)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ev.Body.BlockSignatures, evFromWire.Body.BlockSignatures) {
			t.Fatalf("Error converting %s.Body.BlockSignatures from light wire", k)
		}

		if !reflect.DeepEqual(ev.Body, evFromWire.Body) {
			t.Fatalf("Error converting %s.Body from light wire", k)
		}

		if !reflect.DeepEqual(ev.Signature, evFromWire.Signature) {
			t.Fatalf("Error converting %s.Signature from light wire", k)
		}

		ok, err := ev.Verify()
		if !ok {
			t.Fatalf("Error verifying signature for %s from ligh wire: %v", k, err)
		}
	}
}

func TestStronglySee(t *testing.T) {
	h, index := initRoundHashgraph(t)

	expected := []ancestryItem{
		ancestryItem{"e21", "e0", true, false},
		ancestryItem{"e02", "e10", true, false},
		ancestryItem{"e02", "e0", true, false},
		ancestryItem{"e02", "e1", true, false},
		ancestryItem{"f1", "e21", true, false},
		ancestryItem{"f1", "e10", true, false},
		ancestryItem{"f1", "e0", true, false},
		ancestryItem{"f1", "e1", true, false},
		ancestryItem{"f1", "e2", true, false},
		ancestryItem{"s11", "e2", true, false},
		//false negatives
		ancestryItem{"e10", "e0", false, false},
		ancestryItem{"e21", "e1", false, false},
		ancestryItem{"e21", "e2", false, false},
		ancestryItem{"e02", "e2", false, false},
		ancestryItem{"s11", "e02", false, false},
		ancestryItem{"s11", "", false, true},
	}

	lastPeerSet, err := h.Store.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}

	for _, exp := range expected {
		a, err := h.stronglySee(index[exp.descendant], index[exp.ancestor], lastPeerSet)
		if err != nil && !exp.err {
			t.Fatalf("Error computing stronglySee(%s, %s). Err: %v", exp.descendant, exp.ancestor, err)
		}
		if a != exp.val {
			t.Fatalf("stronglySee(%s, %s) should be %v, not %v", exp.descendant, exp.ancestor, exp.val, a)
		}
	}
}

func TestWitness(t *testing.T) {
	h, index := initRoundHashgraph(t)

	expected := []ancestryItem{
		ancestryItem{"", "e0", true, false},
		ancestryItem{"", "e1", true, false},
		ancestryItem{"", "e2", true, false},
		ancestryItem{"", "f1", true, false},
		ancestryItem{"", "e10", false, false},
		ancestryItem{"", "e21", false, false},
		ancestryItem{"", "e02", false, false},
	}

	for _, exp := range expected {
		a, err := h.witness(index[exp.ancestor])
		if err != nil {
			t.Fatalf("Error computing witness(%s). Err: %v", exp.ancestor, err)
		}
		if a != exp.val {
			t.Fatalf("witness(%s) should be %v, not %v", exp.ancestor, exp.val, a)
		}
	}
}

func TestRound(t *testing.T) {
	h, index := initRoundHashgraph(t)

	expected := []roundItem{
		roundItem{"e0", 0},
		roundItem{"e1", 0},
		roundItem{"e2", 0},
		roundItem{"s00", 0},
		roundItem{"e10", 0},
		roundItem{"s20", 0},
		roundItem{"e21", 0},
		roundItem{"e02", 0},
		roundItem{"s10", 0},
		roundItem{"f1", 1},
		roundItem{"s11", 1},
	}

	for _, exp := range expected {
		r, err := h.round(index[exp.event])
		if err != nil {
			t.Fatalf("Error computing round(%s). Err: %v", exp.event, err)
		}
		if r != exp.round {
			t.Fatalf("round(%s) should be %v, not %v", exp.event, exp.round, r)
		}
	}
}

func TestRoundDiff(t *testing.T) {
	h, index := initRoundHashgraph(t)

	if d, err := h.roundDiff(index["f1"], index["e02"]); d != 1 {
		if err != nil {
			t.Fatalf("RoundDiff(f1, e02) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(f1, e02) should be 1 not %d", d)
	}

	if d, err := h.roundDiff(index["e02"], index["f1"]); d != -1 {
		if err != nil {
			t.Fatalf("RoundDiff(e02, f1) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e02, f1) should be -1 not %d", d)
	}
	if d, err := h.roundDiff(index["e02"], index["e21"]); d != 0 {
		if err != nil {
			t.Fatalf("RoundDiff(e20, e21) returned an error: %s", err)
		}
		t.Fatalf("RoundDiff(e20, e21) should be 0 not %d", d)
	}
}

func TestDivideRounds(t *testing.T) {
	h, index := initRoundHashgraph(t)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}

	if l := h.Store.LastRound(); l != 1 {
		t.Fatalf("last round should be 1 not %d", l)
	}

	expectedRounds := map[int]*RoundInfo{
		0: &RoundInfo{
			CreatedEvents: map[string]RoundEvent{
				index["e0"]:  RoundEvent{Witness: true, Famous: Undefined},
				index["e1"]:  RoundEvent{Witness: true, Famous: Undefined},
				index["e2"]:  RoundEvent{Witness: true, Famous: Undefined},
				index["e10"]: RoundEvent{Witness: false, Famous: Undefined},
				index["s20"]: RoundEvent{Witness: false, Famous: Undefined},
				index["e21"]: RoundEvent{Witness: false, Famous: Undefined},
				index["s00"]: RoundEvent{Witness: false, Famous: Undefined},
				index["e02"]: RoundEvent{Witness: false, Famous: Undefined},
				index["s10"]: RoundEvent{Witness: false, Famous: Undefined},
			},
		},
		1: &RoundInfo{
			CreatedEvents: map[string]RoundEvent{
				index["f1"]:  RoundEvent{Witness: true, Famous: Undefined},
				index["s11"]: RoundEvent{Witness: false, Famous: Undefined},
			},
		},
	}

	round0, err := h.Store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedRounds[0].CreatedEvents, round0.CreatedEvents) {
		t.Fatalf("Round[0].CreatedEvents should be %v, not %v", expectedRounds[0].CreatedEvents, round0.CreatedEvents)
	}

	round1, err := h.Store.GetRound(1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedRounds[1].CreatedEvents, round1.CreatedEvents) {
		t.Fatalf("Round[1].CreatedEvents should be %v, not %v", expectedRounds[1].CreatedEvents, round1.CreatedEvents)
	}

	expectedPendingRounds := []pendingRound{
		pendingRound{
			Index:   0,
			Decided: false,
		},
		pendingRound{
			Index:   1,
			Decided: false,
		},
	}
	for i, pd := range h.PendingRounds {
		if !reflect.DeepEqual(*pd, expectedPendingRounds[i]) {
			t.Fatalf("pendingRounds[%d] should be %v, not %v", i, expectedPendingRounds[i], *pd)
		}
	}

	//[event] => {lamportTimestamp, round}
	type tr struct {
		t, r int
	}
	expectedTimestamps := map[string]tr{
		"e0":  tr{0, 0},
		"e1":  tr{0, 0},
		"e2":  tr{0, 0},
		"s00": tr{1, 0},
		"e10": tr{1, 0},
		"s20": tr{1, 0},
		"e21": tr{2, 0},
		"e02": tr{3, 0},
		"s10": tr{2, 0},
		"f1":  tr{4, 1},
		"s11": tr{5, 1},
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

}

func TestCreateRoot(t *testing.T) {
	h, index := initRoundHashgraph(t)
	h.DivideRounds()

	peerSet, err := h.Store.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]*Root{
		"e0": NewBaseRoot(peerSet.Peers[0].ID),
		"e02": &Root{
			NextRound:  0,
			SelfParent: RootEvent{index["s00"], peerSet.Peers[0].ID, 1, 1, 0},
			Others: map[string]RootEvent{
				index["e02"]: RootEvent{index["e21"], peerSet.Peers[2].ID, 2, 2, 0},
			},
		},
		"s10": &Root{
			NextRound:  0,
			SelfParent: RootEvent{index["e10"], peerSet.Peers[1].ID, 1, 1, 0},
			Others:     map[string]RootEvent{},
		},
		"f1": &Root{
			NextRound:  1,
			SelfParent: RootEvent{index["s10"], peerSet.Peers[1].ID, 2, 2, 0},
			Others: map[string]RootEvent{
				index["f1"]: RootEvent{index["e02"], peerSet.Peers[0].ID, 2, 3, 0},
			},
		},
	}

	for evh, expRoot := range expected {
		ev, err := h.Store.GetEvent(index[evh])
		if err != nil {
			t.Fatal(err)
		}
		root, err := h.createRoot(ev)
		if err != nil {
			t.Fatalf("Error creating %s Root: %v", evh, err)
		}
		if !reflect.DeepEqual(expRoot, root) {
			t.Fatalf("%s Root should be %v, not %v", evh, expRoot, root)
		}
	}

}

func contains(s []string, x string) bool {
	for _, e := range s {
		if e == x {
			return true
		}
	}
	return false
}

/*



e01  e12
 |   |  \
 e0  R1  e2
 |       |
 R0      R2

*/
func initDentedHashgraph(t *testing.T) (*Hashgraph, map[string]string) {
	nodes, index, orderedEvents, participants := initHashgraphNodes(n)

	orderedPeers := participants.Peers

	for _, peer := range orderedPeers {
		index[rootSelfParent(peer.ID)] = rootSelfParent(peer.ID)
	}

	plays := []play{
		play{0, 0, rootSelfParent(orderedPeers[0].ID), "", "e0", nil, nil},
		play{2, 0, rootSelfParent(orderedPeers[2].ID), "", "e2", nil, nil},
		play{0, 1, "e0", "", "e01", nil, nil},
		play{1, 0, rootSelfParent(orderedPeers[1].ID), "e2", "e12", nil, nil},
	}

	playEvents(plays, nodes, index, orderedEvents)

	h := createHashgraph(false, orderedEvents, participants, testLogger(t))

	//Set Rounds manually; this would normally be handled by DivideRounds()
	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e12"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, &RoundInfo{CreatedEvents: round0Witnesses})
	h.Store.SetPeerSet(1, participants)

	return h, index
}

func TestCreateRootBis(t *testing.T) {
	h, index := initDentedHashgraph(t)

	peerSet, _ := h.Store.GetLastPeerSet()

	expected := map[string]*Root{
		"e12": &Root{
			NextRound:  0,
			SelfParent: NewBaseRootEvent(peerSet.Peers[1].ID),
			Others: map[string]RootEvent{
				index["e12"]: RootEvent{index["e2"], peerSet.Peers[2].ID, 0, 0, 0},
			},
		},
	}

	for evh, expRoot := range expected {
		ev, err := h.Store.GetEvent(index[evh])
		if err != nil {
			t.Fatal(err)
		}
		root, err := h.createRoot(ev)
		if err != nil {
			t.Fatalf("Error creating %s Root: %v", evh, err)
		}
		if !reflect.DeepEqual(expRoot, root) {
			t.Fatalf("%s Root should be %v, not %v", evh, expRoot, root)
		}
	}
}

/*

e0  e1  e2    Block (0, 1)
0   1    2
*/
func initBlockHashgraph(t *testing.T) (*Hashgraph, []TestNode, map[string]string) {
	nodes, index, orderedEvents, peerSet := initHashgraphNodes(n)

	for i, peer := range peerSet.Peers {
		event := NewEvent(nil, nil, nil, []string{rootSelfParent(peer.ID), ""}, nodes[i].Pub, 0)
		nodes[i].signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
	}

	hashgraph := NewHashgraph(peerSet, NewInmemStore(peerSet, cacheSize), DummyInternalCommitCallback, testLogger(t))

	//create a block and signatures manually
	block := NewBlock(0, 1,
		[]byte("framehash"),
		peerSet.Peers,
		[][]byte{[]byte("block tx")},
		[]InternalTransaction{
			NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1", "paris")),
			NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2", "london")),
		})

	err := hashgraph.Store.SetBlock(block)
	if err != nil {
		t.Fatalf("Error setting block. Err: %s", err)
	}

	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev, true); err != nil {
			t.Fatalf("ERROR inserting event %d: %s\n", i, err)
		}
	}

	return hashgraph, nodes, index
}

func TestInsertEventsWithBlockSignatures(t *testing.T) {
	h, nodes, index := initBlockHashgraph(t)

	block, err := h.Store.GetBlock(0)
	if err != nil {
		t.Fatalf("Error retrieving block 0. %s", err)
	}

	blockSigs := make([]BlockSignature, n)
	for k, n := range nodes {
		blockSigs[k], err = block.Sign(n.Key)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("Inserting Events with valid signatures", func(t *testing.T) {

		/*
			s00 |   |
			|   |   |
			|  e10  s20
			| / |   |
			e0  e1  e2
			0   1    2
		*/
		plays := []play{
			play{1, 1, "e1", "e0", "e10", nil, []BlockSignature{blockSigs[1]}},
			play{2, 1, "e2", "", "s20", nil, []BlockSignature{blockSigs[2]}},
			play{0, 1, "e0", "", "s00", nil, []BlockSignature{blockSigs[0]}},
		}

		for _, p := range plays {
			e := NewEvent(p.txPayload,
				nil,
				p.sigPayload,
				[]string{index[p.selfParent], index[p.otherParent]},
				nodes[p.to].Pub,
				p.index)
			e.Sign(nodes[p.to].Key)
			index[p.name] = e.Hex()
			if err := h.InsertEvent(e, true); err != nil {
				t.Fatalf("ERROR inserting event %s: %s\n", p.name, err)
			}
		}

		//Check SigPool
		if l := len(h.SigPool); l != 3 {
			t.Fatalf("SigPool should contain 3 signatures, not %d", l)
		}

		//Process SigPool
		h.ProcessSigPool()

		//Check that the block contains 3 signatures
		block, _ := h.Store.GetBlock(0)
		if l := len(block.Signatures); l != 3 {
			t.Fatalf("Block 0 should contain 3 signatures, not %d", l)
		}

		//Check that SigPool was cleared
		if l := len(h.SigPool); l != 0 {
			t.Fatalf("SigPool should contain 0 signatures, not %d", l)
		}
	})

	t.Run("Inserting Events with signature of unknown block", func(t *testing.T) {
		//The Event should be inserted
		//The block signature is simply ignored
		peerSet, err := h.Store.GetPeerSet(2)
		if err != nil {
			t.Fatal(err)
		}
		block1 := NewBlock(1, 2, []byte("framehash"), peerSet.Peers, [][]byte{}, []InternalTransaction{})
		sig, _ := block1.Sign(nodes[2].Key)

		//unknown block
		unknownBlockSig := BlockSignature{
			Validator: nodes[2].Pub,
			Index:     1,
			Signature: sig.Signature,
		}
		p := play{2, 2, "s20", "e10", "e21", nil, []BlockSignature{unknownBlockSig}}

		e := NewEvent(nil,
			nil,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		e.Sign(nodes[p.to].Key)
		index[p.name] = e.Hex()
		if err := h.InsertEvent(e, true); err != nil {
			t.Fatalf("ERROR inserting event %s: %s", p.name, err)
		}

		//check that the event was recorded
		_, err = h.Store.GetEvent(index["e21"])
		if err != nil {
			t.Fatalf("ERROR fetching Event e21: %s", err)
		}

	})

	t.Run("Inserting Events with BlockSignature not from creator", func(t *testing.T) {
		//The Event should be inserted
		//The block signature is simply ignored

		//wrong validator
		//Validator should be same as Event creator (node 0)
		key, _ := crypto.GenerateECDSAKey()
		badNode := NewTestNode(key)
		badNodeSig, _ := block.Sign(badNode.Key)

		p := play{0, 2, "s00", "e21", "e02", nil, []BlockSignature{badNodeSig}}

		e := NewEvent(nil,
			nil,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		e.Sign(nodes[p.to].Key)
		index[p.name] = e.Hex()
		if err := h.InsertEvent(e, true); err != nil {
			t.Fatalf("ERROR inserting event %s: %s\n", p.name, err)
		}

		//check that the signature was not appended to the block
		block, _ := h.Store.GetBlock(0)
		if l := len(block.Signatures); l > 3 {
			t.Fatalf("Block 0 should contain 3 signatures, not %d", l)
		}
	})

}

/*
                  Round 4
		i0  |   i2
		| \ | / |
		|   i1  |
------- |  /|   | --------------------------------
		h02 |   | Round 3
		| \ |   |
		|   \   |
		|   | \ |
		|   |  h21
		|   | / |
		|  h10  |
		| / |   |
		h0  |   h2
		| \ | / |
		|   h1  |
------- |  /|   | --------------------------------
		g02 |   | Round 2
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
------- |  /|   | -------------------------------
		f02b|   |  Round 1           +---------+
		|   |   |                    | Block 1 |
		f02 |   |                    | RR    2 |
		| \ |   |                    | Evs   9 |
		|   \   |                    +---------+
		|   | \ |
	---f0x  |   f21 //f0x's other-parent is e21. This situation can happen with concurrency
	|	|   | / |
	|	|  f10  |
	|	| / |   |
	|	f0  |   f2
	|	| \ | / |
	|	|  f1b  |
	|	|   |   |
	|	|   f1  |
---	| -	|  /|   | ------------------------------
	|	e02 |   |  Round 0          +---------+
	|	| \ |   |                   | Block 0 |
	|	|   \   |                   | RR    1 |
	|	|   | \ |                   | Evs   7 |
	|   |   | e21b                  +---------+
	|	|   |   |
	---------- e21
		|   | / |
		|  e10  |
	    | / |   |
		e0  e1  e2
		0   1    2
*/
func initConsensusHashgraph(db bool, t testing.TB) (*Hashgraph, map[string]string) {
	plays := []play{
		play{1, 1, "e1", "e0", "e10", nil, nil},
		play{2, 1, "e2", "e10", "e21", [][]byte{[]byte("e21")}, nil},
		play{2, 2, "e21", "", "e21b", nil, nil},
		play{0, 1, "e0", "e21b", "e02", nil, nil},
		play{1, 2, "e10", "e02", "f1", nil, nil},
		play{1, 3, "f1", "", "f1b", [][]byte{[]byte("f1b")}, nil},
		play{0, 2, "e02", "f1b", "f0", nil, nil},
		play{2, 3, "e21b", "f1b", "f2", nil, nil},
		play{1, 4, "f1b", "f0", "f10", nil, nil},
		play{0, 3, "f0", "e21", "f0x", nil, nil},
		play{2, 4, "f2", "f10", "f21", nil, nil},
		play{0, 4, "f0x", "f21", "f02", nil, nil},
		play{0, 5, "f02", "", "f02b", [][]byte{[]byte("f02b")}, nil},
		play{1, 5, "f10", "f02b", "g1", nil, nil},
		play{0, 6, "f02b", "g1", "g0", nil, nil},
		play{2, 5, "f21", "g1", "g2", nil, nil},
		play{1, 6, "g1", "g0", "g10", [][]byte{[]byte("g10")}, nil},
		play{2, 6, "g2", "g10", "g21", nil, nil},
		play{0, 7, "g0", "g21", "g02", [][]byte{[]byte("g02")}, nil},
		play{1, 7, "g10", "g02", "h1", nil, nil},
		play{0, 8, "g02", "h1", "h0", nil, nil},
		play{2, 7, "g21", "h1", "h2", nil, nil},
		play{1, 8, "h1", "h0", "h10", nil, nil},
		play{2, 8, "h2", "h10", "h21", nil, nil},
		play{0, 9, "h0", "h21", "h02", nil, nil},
		play{1, 9, "h10", "h02", "i1", nil, nil},
		play{0, 10, "h02", "i1", "i0", nil, nil},
		play{2, 9, "h21", "i1", "i2", nil, nil},
	}

	hashgraph, index, _ := initHashgraphFull(plays, db, n, testLogger(t))

	return hashgraph, index
}

func TestDivideRoundsBis(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}

	expectedCreatedEvents := map[int]map[string]RoundEvent{
		0: map[string]RoundEvent{
			index["e0"]:   RoundEvent{Witness: true, Famous: Undefined},
			index["e1"]:   RoundEvent{Witness: true, Famous: Undefined},
			index["e2"]:   RoundEvent{Witness: true, Famous: Undefined},
			index["e10"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["e21"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["e21b"]: RoundEvent{Witness: false, Famous: Undefined},
			index["e02"]:  RoundEvent{Witness: false, Famous: Undefined},
		},
		1: map[string]RoundEvent{
			index["f1"]:   RoundEvent{Witness: true, Famous: Undefined},
			index["f1b"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f0"]:   RoundEvent{Witness: true, Famous: Undefined},
			index["f2"]:   RoundEvent{Witness: true, Famous: Undefined},
			index["f10"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f21"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f0x"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f02"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f02b"]: RoundEvent{Witness: false, Famous: Undefined},
		},
		2: map[string]RoundEvent{
			index["g1"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["g0"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["g2"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["g10"]: RoundEvent{Witness: false, Famous: Undefined},
			index["g21"]: RoundEvent{Witness: false, Famous: Undefined},
			index["g02"]: RoundEvent{Witness: false, Famous: Undefined},
		},
		3: map[string]RoundEvent{
			index["h1"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["h0"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["h2"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["h10"]: RoundEvent{Witness: false, Famous: Undefined},
			index["h21"]: RoundEvent{Witness: false, Famous: Undefined},
			index["h02"]: RoundEvent{Witness: false, Famous: Undefined},
		},
		4: map[string]RoundEvent{
			index["i1"]: RoundEvent{Witness: true, Famous: Undefined},
			index["i0"]: RoundEvent{Witness: true, Famous: Undefined},
			index["i2"]: RoundEvent{Witness: true, Famous: Undefined},
		},
	}

	for i := 0; i < 5; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expectedCreatedEvents[i], round.CreatedEvents) {
			t.Fatalf("Round[%d].CreatedEvents should be %v, not %v", i, expectedCreatedEvents[i], round.CreatedEvents)
		}
	}

	//[event] => {lamportTimestamp, round}
	type tr struct {
		t, r int
	}
	expectedTimestamps := map[string]tr{
		"e0":   tr{0, 0},
		"e1":   tr{0, 0},
		"e2":   tr{0, 0},
		"e10":  tr{1, 0},
		"e21":  tr{2, 0},
		"e21b": tr{3, 0},
		"e02":  tr{4, 0},
		"f1":   tr{5, 1},
		"f1b":  tr{6, 1},
		"f0":   tr{7, 1},
		"f2":   tr{7, 1},
		"f10":  tr{8, 1},
		"f0x":  tr{8, 1},
		"f21":  tr{9, 1},
		"f02":  tr{10, 1},
		"f02b": tr{11, 1},
		"g1":   tr{12, 2},
		"g0":   tr{13, 2},
		"g2":   tr{13, 2},
		"g10":  tr{14, 2},
		"g21":  tr{15, 2},
		"g02":  tr{16, 2},
		"h1":   tr{17, 3},
		"h0":   tr{18, 3},
		"h2":   tr{18, 3},
		"h10":  tr{19, 3},
		"h21":  tr{20, 3},
		"h02":  tr{21, 3},
		"i1":   tr{22, 4},
		"i0":   tr{23, 4},
		"i2":   tr{23, 4},
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

}

func TestDecideFame(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	h.DivideRounds()
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}

	expectedCreatedEvents := map[int]map[string]RoundEvent{
		0: map[string]RoundEvent{
			index["e0"]:   RoundEvent{Witness: true, Famous: True},
			index["e1"]:   RoundEvent{Witness: true, Famous: True},
			index["e2"]:   RoundEvent{Witness: true, Famous: True},
			index["e10"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["e21"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["e21b"]: RoundEvent{Witness: false, Famous: Undefined},
			index["e02"]:  RoundEvent{Witness: false, Famous: Undefined},
		},
		1: map[string]RoundEvent{
			index["f1"]:   RoundEvent{Witness: true, Famous: True},
			index["f1b"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f0"]:   RoundEvent{Witness: true, Famous: True},
			index["f2"]:   RoundEvent{Witness: true, Famous: True},
			index["f10"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f21"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f0x"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f02"]:  RoundEvent{Witness: false, Famous: Undefined},
			index["f02b"]: RoundEvent{Witness: false, Famous: Undefined},
		},
		2: map[string]RoundEvent{
			index["g1"]:  RoundEvent{Witness: true, Famous: True},
			index["g0"]:  RoundEvent{Witness: true, Famous: True},
			index["g2"]:  RoundEvent{Witness: true, Famous: True},
			index["g10"]: RoundEvent{Witness: false, Famous: Undefined},
			index["g21"]: RoundEvent{Witness: false, Famous: Undefined},
			index["g02"]: RoundEvent{Witness: false, Famous: Undefined},
		},
		3: map[string]RoundEvent{
			index["h1"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["h0"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["h2"]:  RoundEvent{Witness: true, Famous: Undefined},
			index["h10"]: RoundEvent{Witness: false, Famous: Undefined},
			index["h21"]: RoundEvent{Witness: false, Famous: Undefined},
			index["h02"]: RoundEvent{Witness: false, Famous: Undefined},
		},
		4: map[string]RoundEvent{
			index["i1"]: RoundEvent{Witness: true, Famous: Undefined},
			index["i0"]: RoundEvent{Witness: true, Famous: Undefined},
			index["i2"]: RoundEvent{Witness: true, Famous: Undefined},
		},
	}

	for i := 0; i < 5; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expectedCreatedEvents[i], round.CreatedEvents) {
			t.Fatalf("Round[%d].CreatedEvents should be %v, not %v", i, expectedCreatedEvents[i], round.CreatedEvents)
		}
	}

	expectedpendingRounds := []pendingRound{
		pendingRound{
			Index:   0,
			Decided: true,
		},
		pendingRound{
			Index:   1,
			Decided: true,
		},
		pendingRound{
			Index:   2,
			Decided: true,
		},
		pendingRound{
			Index:   3,
			Decided: false,
		},
		pendingRound{
			Index:   4,
			Decided: false,
		},
	}
	for i, pd := range h.PendingRounds {
		if !reflect.DeepEqual(*pd, expectedpendingRounds[i]) {
			t.Fatalf("pendingRounds[%d] should be %v, not %v", i, expectedpendingRounds[i], *pd)
		}
	}
}

func TestDecideRoundReceived(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	h.DivideRounds()
	h.DecideFame()
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}

	expectedReceivedEvents := map[int][]string{
		0: []string{},
		1: []string{index["e0"], index["e1"], index["e2"], index["e10"], index["e21"], index["e21b"], index["e02"]},
		2: []string{index["f1"], index["f1b"], index["f0"], index["f2"], index["f10"], index["f0x"], index["f21"], index["f02"], index["f02b"]},
		3: []string{},
		4: []string{},
	}

	for i := 0; i < 5; i++ {
		round, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expectedReceivedEvents[i], round.ReceivedEvents) {
			t.Fatalf("Round[%d].ReceivedEvents should be %v, not %v", i, expectedReceivedEvents[i], round.ReceivedEvents)
		}
	}

	for name, hash := range index {
		e, _ := h.Store.GetEvent(hash)
		if rune(name[0]) == rune('e') {
			if r := *e.roundReceived; r != 1 {
				t.Fatalf("%s round received should be 1 not %d", name, r)
			}
		} else if rune(name[0]) == rune('f') {
			if r := *e.roundReceived; r != 2 {
				t.Fatalf("%s round received should be 2 not %d", name, r)
			}
		} else if e.roundReceived != nil {
			t.Fatalf("%s round received should be nil not %d", name, *e.roundReceived)
		}
	}

	expectedUndeterminedEvents := []string{
		index["g1"],
		index["g0"],
		index["g2"],
		index["g10"],
		index["g21"],
		index["g02"],
		index["h1"],
		index["h0"],
		index["h2"],
		index["h10"],
		index["h21"],
		index["h02"],
		index["i1"],
		index["i0"],
		index["i2"],
	}

	for i, eue := range expectedUndeterminedEvents {
		if ue := h.UndeterminedEvents[i]; ue != eue {
			t.Fatalf("UndeterminedEvents[%d] should be %s, not %s", i, eue, ue)
		}
	}
}

func TestProcessDecidedRounds(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

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

	if l := len(consensusEvents); l != 16 {
		t.Fatalf("length of consensus should be 16 not %d", l)
	}

	if ple := h.PendingLoadedEvents; ple != 2 {
		t.Fatalf("PendingLoadedEvents should be 2, not %d", ple)
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

	if l := len(block0.Transactions()); l != 1 {
		t.Fatalf("Block0 should contain 1 transaction, not %d", l)
	}
	if tx := block0.Transactions()[0]; !reflect.DeepEqual(tx, []byte("e21")) {
		t.Fatalf("Block0.Transactions[0] should be 'e21', not %s", tx)
	}

	frame1, err := h.GetFrame(block0.RoundReceived())
	frame1Hash, err := frame1.Hash()
	if !reflect.DeepEqual(block0.FrameHash(), frame1Hash) {
		t.Fatalf("Block0.FrameHash should be %v, not %v", frame1Hash, block0.FrameHash())
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

	if l := len(block1.Transactions()); l != 2 {
		t.Fatalf("Block1 should contain 2 transactions, not %d", l)
	}
	if tx := block1.Transactions()[1]; !reflect.DeepEqual(tx, []byte("f02b")) {
		t.Fatalf("Block1.Transactions[1] should be 'f02b', not %s", tx)
	}

	frame2, err := h.GetFrame(block1.RoundReceived())
	frame2Hash, err := frame2.Hash()
	if !reflect.DeepEqual(block1.FrameHash(), frame2Hash) {
		t.Fatalf("Block1.FrameHash should be %v, not %v", frame2Hash, block1.FrameHash())
	}

	// pendingRounds -----------------------------------------------------------
	expectedpendingRounds := []pendingRound{
		pendingRound{
			Index:   3,
			Decided: false,
		},
		pendingRound{
			Index:   4,
			Decided: false,
		},
	}
	for i, pd := range h.PendingRounds {
		if !reflect.DeepEqual(*pd, expectedpendingRounds[i]) {
			t.Fatalf("pendingRounds[%d] should be %v, not %v", i, expectedpendingRounds[i], *pd)
		}
	}

	//Anchor -------------------------------------------------------------------
	if v := h.AnchorBlock; v != nil {
		t.Fatalf("AnchorBlock should be nil, not %v", v)
	}

}

func BenchmarkConsensus(b *testing.B) {
	for n := 0; n < b.N; n++ {
		//we do not want to benchmark the initialization code
		b.StopTimer()
		h, _ := initConsensusHashgraph(false, b)
		b.StartTimer()

		h.DivideRounds()
		h.DecideFame()
		h.DecideRoundReceived()
		h.ProcessDecidedRounds()
	}
}

func TestKnown(t *testing.T) {
	h, _ := initConsensusHashgraph(false, t)

	peerSet, _ := h.Store.GetLastPeerSet()

	expectedKnown := map[int]int{
		peerSet.IDs()[0]: 10,
		peerSet.IDs()[1]: 9,
		peerSet.IDs()[2]: 9,
	}

	known := h.Store.KnownEvents()
	for i := range peerSet.IDs() {
		if l := known[i]; l != expectedKnown[i] {
			t.Fatalf("Known[%d] should be %d, not %d", i, expectedKnown[i], l)
		}
	}
}

func TestGetFrame(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	peerSet, _ := h.Store.GetLastPeerSet()

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	h.ProcessDecidedRounds()

	t.Run("Round 1", func(t *testing.T) {
		expectedRoots := make(map[string]*Root, n)
		expectedRoots[peerSet.PubKeys()[0]] = NewBaseRoot(peerSet.IDs()[0])
		expectedRoots[peerSet.PubKeys()[1]] = NewBaseRoot(peerSet.IDs()[1])
		expectedRoots[peerSet.PubKeys()[2]] = NewBaseRoot(peerSet.IDs()[2])

		frame, err := h.GetFrame(1)
		if err != nil {
			t.Fatal(err)
		}

		for p, r := range frame.Roots {
			er := expectedRoots[p]
			if x := r.SelfParent; !reflect.DeepEqual(x, er.SelfParent) {
				t.Fatalf("Roots[%v].SelfParent should be %v, not %v", p, er.SelfParent, x)
			}
			if others := r.Others; !reflect.DeepEqual(others, er.Others) {
				t.Fatalf("Roots[%v].Others should be %v, not %v", p, er.Others, others)
			}
		}

		expectedEventsHashes := []string{
			index["e0"],
			index["e1"],
			index["e2"],
			index["e10"],
			index["e21"],
			index["e21b"],
			index["e02"]}
		expectedEvents := []*Event{}
		for _, eh := range expectedEventsHashes {
			e, err := h.Store.GetEvent(eh)
			if err != nil {
				t.Fatal(err)
			}
			expectedEvents = append(expectedEvents, e)
		}
		sort.Sort(ByLamportTimestamp(expectedEvents))
		if !reflect.DeepEqual(expectedEvents, frame.Events) {
			t.Fatal("Frame.Events is not good")
		}

		block0, err := h.Store.GetBlock(0)
		if err != nil {
			t.Fatalf("Store should contain a block with Index 1: %v", err)
		}
		frame1Hash, err := frame.Hash()
		if err != nil {
			t.Fatalf("Error computing Frame hash, %v", err)
		}
		if !reflect.DeepEqual(block0.FrameHash(), frame1Hash) {
			t.Fatalf("Block0.FrameHash (%v) and Frame1.Hash (%v) differ", block0.FrameHash(), frame1Hash)
		}
	})

	t.Run("Round 2", func(t *testing.T) {
		expectedRoots := make(map[string]*Root, n)
		expectedRoots[peerSet.PubKeys()[0]] = &Root{
			NextRound:  1,
			SelfParent: RootEvent{index["e02"], peerSet.IDs()[0], 1, 4, 0},
			Others: map[string]RootEvent{
				index["f0"]: RootEvent{
					Hash:             index["f1b"],
					CreatorID:        peerSet.IDs()[1],
					Index:            3,
					LamportTimestamp: 6,
					Round:            1,
				},
				index["f0x"]: RootEvent{
					Hash:             index["e21"],
					CreatorID:        peerSet.IDs()[2],
					Index:            1,
					LamportTimestamp: 2,
					Round:            0,
				},
			},
		}
		expectedRoots[peerSet.PubKeys()[1]] = &Root{
			NextRound:  1,
			SelfParent: RootEvent{index["e10"], peerSet.IDs()[1], 1, 1, 0},
			Others: map[string]RootEvent{
				index["f1"]: RootEvent{
					Hash:             index["e02"],
					CreatorID:        peerSet.IDs()[0],
					Index:            1,
					LamportTimestamp: 4,
					Round:            0,
				},
			},
		}
		expectedRoots[peerSet.PubKeys()[2]] = &Root{
			NextRound:  1,
			SelfParent: RootEvent{index["e21b"], peerSet.IDs()[2], 2, 3, 0},
			Others: map[string]RootEvent{
				index["f2"]: RootEvent{
					Hash:             index["f1b"],
					CreatorID:        peerSet.IDs()[1],
					Index:            3,
					LamportTimestamp: 6,
					Round:            1,
				},
			},
		}

		frame, err := h.GetFrame(2)
		if err != nil {
			t.Fatal(err)
		}

		for p, r := range frame.Roots {
			er := expectedRoots[p]
			if x := r.SelfParent; !reflect.DeepEqual(x, er.SelfParent) {
				t.Fatalf("Roots[%v].SelfParent should be %v, not %v", p, er.SelfParent, x)
			}

			if others := r.Others; !reflect.DeepEqual(others, er.Others) {
				t.Fatalf("Roots[%v].Others should be %v, not %v", p, er.Others, others)
			}
		}

		expectedEventsHashes := []string{
			index["f1"],
			index["f1b"],
			index["f0"],
			index["f2"],
			index["f10"],
			index["f0x"],
			index["f21"],
			index["f02"],
			index["f02b"]}
		expectedEvents := []*Event{}
		for _, eh := range expectedEventsHashes {
			e, err := h.Store.GetEvent(eh)
			if err != nil {
				t.Fatal(err)
			}
			expectedEvents = append(expectedEvents, e)
		}
		sort.Sort(ByLamportTimestamp(expectedEvents))
		if !reflect.DeepEqual(expectedEvents, frame.Events) {
			t.Fatal("Frame.Events is not good")
		}
	})

}

func TestResetFromFrame(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	peerSet, _ := h.Store.GetLastPeerSet()

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	h.ProcessDecidedRounds()

	block, err := h.Store.GetBlock(1)
	if err != nil {
		t.Fatal(err)
	}

	frame, err := h.GetFrame(block.RoundReceived())
	if err != nil {
		t.Fatal(err)
	}

	//This operation clears the private fields which need to be recomputed
	//in the Events (round, roundReceived,etc)
	marshalledFrame, _ := frame.Marshal()
	unmarshalledFrame := new(Frame)
	unmarshalledFrame.Unmarshal(marshalledFrame)

	h2 := NewHashgraph(peerSet,
		NewInmemStore(peerSet, cacheSize),
		DummyInternalCommitCallback,
		testLogger(t))
	err = h2.Reset(block, unmarshalledFrame)
	if err != nil {
		t.Fatal(err)
	}

	/*
		The hashgraph should now look like this:

		   	   f02b|   |
		   	   |   |   |
		   	   f02 |   |
		   	   | \ |   |
		   	   |   \   |
		   	   |   | \ |
		   +--f0x  |   f21 //f0x's other-parent is e21b; contained in R0
		   |   |   | / |
		   |   |  f10  |
		   |   | / |   |
		   |   f0  |   f2
		   |   | \ | / |
		   |   |  f1b  |
		   |   |   |   |
		   |   |   f1  |
		   |   |   |   |
		   +-- R0  R1  R2
	*/

	//Test Known
	expectedKnown := map[int]int{
		peerSet.IDs()[0]: 5,
		peerSet.IDs()[1]: 4,
		peerSet.IDs()[2]: 4,
	}

	known := h2.Store.KnownEvents()
	for _, peer := range peerSet.ByID {
		if l := known[peer.ID]; l != expectedKnown[peer.ID] {
			t.Fatalf("Known[%d] should be %d, not %d", peer.ID, expectedKnown[peer.ID], l)
		}
	}

	/***************************************************************************
	 Test DivideRounds
	***************************************************************************/
	if err := h2.DivideRounds(); err != nil {
		t.Fatal(err)
	}

	hRound1, err := h.Store.GetRound(1)
	if err != nil {
		t.Fatal(err)
	}
	h2Round1, err := h2.Store.GetRound(1)
	if err != nil {
		t.Fatal(err)
	}

	//Check Round1 Witnesses
	hWitnesses := hRound1.Witnesses()
	h2Witnesses := h2Round1.Witnesses()
	sort.Strings(hWitnesses)
	sort.Strings(h2Witnesses)
	if !reflect.DeepEqual(hWitnesses, h2Witnesses) {
		t.Fatalf("Reset Hg Round 1 witnesses should be %v, not %v", hWitnesses, h2Witnesses)
	}

	//check Event Rounds and LamportTimestamps
	for _, ev := range frame.Events {
		h2r, err := h2.round(ev.Hex())
		if err != nil {
			t.Fatalf("Error computing %s Round: %d", getName(index, ev.Hex()), h2r)
		}
		hr, _ := h.round(ev.Hex())
		if h2r != hr {

			t.Fatalf("h2[%v].Round should be %d, not %d", getName(index, ev.Hex()), hr, h2r)
		}

		h2s, err := h2.lamportTimestamp(ev.Hex())
		if err != nil {
			t.Fatalf("Error computing %s LamportTimestamp: %d", getName(index, ev.Hex()), h2s)
		}
		hs, _ := h.lamportTimestamp(ev.Hex())
		if h2s != hs {
			t.Fatalf("h2[%v].LamportTimestamp should be %d, not %d", getName(index, ev.Hex()), hs, h2s)
		}
	}

	/***************************************************************************
	Test Consensus
	***************************************************************************/
	h2.DecideFame()
	h2.DecideRoundReceived()
	h2.ProcessDecidedRounds()

	if lbi := h2.Store.LastBlockIndex(); lbi != block.Index() {
		t.Fatalf("LastBlockIndex should be %d, not %d", block.Index(), lbi)
	}

	if r := h2.LastConsensusRound; r == nil || *r != block.RoundReceived() {
		t.Fatalf("LastConsensusRound should be %d, not %d", block.RoundReceived(), *r)
	}

	if v := h2.AnchorBlock; v != nil {
		t.Fatalf("AnchorBlock should be nil, not %v", v)
	}

	/***************************************************************************
	Test continue after Reset
	***************************************************************************/
	//Insert remaining Events into the Reset hashgraph
	for r := 2; r <= 4; r++ {
		round, err := h.Store.GetRound(r)
		if err != nil {
			t.Fatal(err)
		}

		events := []*Event{}
		for e := range round.CreatedEvents {
			ev, err := h.Store.GetEvent(e)
			if err != nil {
				t.Fatal(err)
			}
			events = append(events, ev)
			t.Logf("R%d %s", r, getName(index, e))
		}

		sort.Sort(ByTopologicalOrder(events))

		for _, ev := range events {
			marshalledEv, _ := ev.Marshal()
			unmarshalledEv := new(Event)
			unmarshalledEv.Unmarshal(marshalledEv)

			err = h2.InsertEvent(unmarshalledEv, true)
			if err != nil {
				t.Fatalf("ERR Inserting Event %s: %v", getName(index, ev.Hex()), err)
			}
		}
	}

	if err := h2.DivideRounds(); err != nil {
		t.Fatal(err)
	}
	if err := h2.DecideFame(); err != nil {
		t.Fatal(err)
	}
	if err := h2.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}
	if err := h2.ProcessDecidedRounds(); err != nil {
		t.Fatal(err)
	}

	for r := 1; r <= 4; r++ {
		hRound, err := h.Store.GetRound(r)
		if err != nil {
			t.Fatal(err)
		}
		h2Round, err := h2.Store.GetRound(r)
		if err != nil {
			t.Fatal(err)
		}

		hWitnesses := hRound.Witnesses()
		h2Witnesses := h2Round.Witnesses()
		sort.Strings(hWitnesses)
		sort.Strings(h2Witnesses)

		if !reflect.DeepEqual(hWitnesses, h2Witnesses) {
			t.Fatalf("Reset Hg Round %d witnesses should be %v, not %v", r, hWitnesses, h2Witnesses)
		}
	}
}

func TestBootstrap(t *testing.T) {

	//Initialize a first Hashgraph with a DB backend
	//Add events and run consensus methods on it
	h, _ := initConsensusHashgraph(true, t)
	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	h.ProcessDecidedRounds()

	h.Store.Close()
	defer os.RemoveAll(badgerDir)

	//Now we want to create a new Hashgraph based on the database of the previous
	//Hashgraph and see if we can boostrap it to the same state.
	recycledStore, err := LoadBadgerStore(cacheSize, badgerDir)
	peerSet, err := recycledStore.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}
	nh := NewHashgraph(peerSet,
		recycledStore,
		DummyInternalCommitCallback,
		logrus.New().WithField("id", "bootstrapped"))
	err = nh.Bootstrap()
	if err != nil {
		t.Fatal(err)
	}

	hConsensusEvents := h.Store.ConsensusEvents()
	nhConsensusEvents := nh.Store.ConsensusEvents()
	if len(hConsensusEvents) != len(nhConsensusEvents) {
		t.Fatalf("Bootstrapped hashgraph should contain %d consensus events,not %d",
			len(hConsensusEvents), len(nhConsensusEvents))
	}

	hKnown := h.Store.KnownEvents()
	nhKnown := nh.Store.KnownEvents()
	if !reflect.DeepEqual(hKnown, nhKnown) {
		t.Fatalf("Bootstrapped hashgraph's Known should be %#v, not %#v",
			hKnown, nhKnown)
	}

	if *h.LastConsensusRound != *nh.LastConsensusRound {
		t.Fatalf("Bootstrapped hashgraph's LastConsensusRound should be %#v, not %#v",
			*h.LastConsensusRound, *nh.LastConsensusRound)
	}

	if h.LastCommitedRoundEvents != nh.LastCommitedRoundEvents {
		t.Fatalf("Bootstrapped hashgraph's LastCommitedRoundEvents should be %#v, not %#v",
			h.LastCommitedRoundEvents, nh.LastCommitedRoundEvents)
	}

	if h.ConsensusTransactions != nh.ConsensusTransactions {
		t.Fatalf("Bootstrapped hashgraph's ConsensusTransactions should be %#v, not %#v",
			h.ConsensusTransactions, nh.ConsensusTransactions)
	}

	if h.PendingLoadedEvents != nh.PendingLoadedEvents {
		t.Fatalf("Bootstrapped hashgraph's PendingLoadedEvents should be %#v, not %#v",
			h.PendingLoadedEvents, nh.PendingLoadedEvents)
	}
}

/*

	The fame of witness w00 is only decided at round 5.

--------------------------------------------------------------------------------
	|   w51   |    | This section is only added in 'full' mode
    |    |  \ |    | w51 collects votes from w40, w41, w42, and w43. It DECIDES
	|    |   e23   | yes.
----------------\---------------------------------------------------------------
	|    |    |   w43
	|    |    | /  | Round 4 is a Coin Round [(4 -0) mod 4 = 0].
    |    |   w42   | No decision will be made.
    |    | /  |    | w40 collects votes from w33, w32 and w31. It votes yes.
    |   w41   |    | w41 collects votes from w33, w32 and w31. It votes yes.
	| /  |    |    | w42 collects votes from w30, w31, w32 and w33. It votes yes.
   w40   |    |    | w43 collects votes from w30, w31, w32 and w33. It votes yes.
    | \  |    |    |------------------------
    |   d13   |    | w30 collects votes from w20, w21, w22 and w23. It votes yes
    |    |  \ |    | w31 collects votes from w21, w22 and w23. It votes no
   w30   |    \    | w32 collects votes from w20, w21, w22 and w23. It votes yes
    | \  |    | \  | w33 collects votes from w20, w21, w22 and w23. It votes yes
    |   \     |   w33
    |    | \  |  / |Again, none of the witnesses in round 3 are able to decide.
    |    |   w32   |However, a strong majority votes yes
    |    |  / |    |
	|   w31   |    |
    |  / |    |    |--------------------------
   w20   |    |    | w23 collects votes from w11, w12 and w13. It votes no
    |  \ |    |    | w21 collects votes from w11, w12, and w13. It votes no
    |    \    |    | w22 collects votes from w11, w12, w13 and w14. It votes yes
    |    | \  |    | w20 collects votes from w11, w12, w13 and w14. It votes yes
    |    |   w22   |
    |    | /  |    | None of the witnesses in round 2 were able to decide.
    |   c10   |    | They voted according to the majority of votes they observed
    | /  |    |    | in round 1. The vote is split 2-2
   b00  w21   |    |
    |    |  \ |    |
    |    |    \    |
    |    |    | \  |
    |    |    |   w23
    |    |    | /  |------------------------
   w10   |   b21   |
	| \  | /  |    | w10 votes yes (it can see w00)
    |   w11   |    | w11 votes yes
    |    |  \ |    | w12 votes no  (it cannot see w00)
	|    |   w12   | w13 votes no
    |    |    | \  |
    |    |    |   w13
    |    |    | /  |------------------------
    |   a10  a21   | We want to decide the fame of w00
    |  / |  / |    |
    |/  a12   |    |
   a00   |  \ |    |
	|    |   a23   |
    |    |    | \  |
   w00  w01  w02  w03
	0	 1	  2	   3
*/

func initFunkyHashgraph(logger *logrus.Logger, full bool) (*Hashgraph, map[string]string) {
	nodes, index, orderedEvents, participants := initHashgraphNodes(4)

	for i, peer := range participants.Peers {
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, nil, []string{rootSelfParent(peer.ID), ""}, nodes[i].Pub, 0)
		nodes[i].signAndAddEvent(event, name, index, orderedEvents)
	}

	plays := []play{
		play{2, 1, "w02", "w03", "a23", [][]byte{[]byte("a23")}, nil},
		play{1, 1, "w01", "a23", "a12", [][]byte{[]byte("a12")}, nil},
		play{0, 1, "w00", "", "a00", [][]byte{[]byte("a00")}, nil},
		play{1, 2, "a12", "a00", "a10", [][]byte{[]byte("a10")}, nil},
		play{2, 2, "a23", "a12", "a21", [][]byte{[]byte("a21")}, nil},
		play{3, 1, "w03", "a21", "w13", [][]byte{[]byte("w13")}, nil},
		play{2, 3, "a21", "w13", "w12", [][]byte{[]byte("w12")}, nil},
		play{1, 3, "a10", "w12", "w11", [][]byte{[]byte("w11")}, nil},
		play{0, 2, "a00", "w11", "w10", [][]byte{[]byte("w10")}, nil},
		play{2, 4, "w12", "w11", "b21", [][]byte{[]byte("b21")}, nil},
		play{3, 2, "w13", "b21", "w23", [][]byte{[]byte("w23")}, nil},
		play{1, 4, "w11", "w23", "w21", [][]byte{[]byte("w21")}, nil},
		play{0, 3, "w10", "", "b00", [][]byte{[]byte("b00")}, nil},
		play{1, 5, "w21", "b00", "c10", [][]byte{[]byte("c10")}, nil},
		play{2, 5, "b21", "c10", "w22", [][]byte{[]byte("w22")}, nil},
		play{0, 4, "b00", "w22", "w20", [][]byte{[]byte("w20")}, nil},
		play{1, 6, "c10", "w20", "w31", [][]byte{[]byte("w31")}, nil},
		play{2, 6, "w22", "w31", "w32", [][]byte{[]byte("w32")}, nil},
		play{0, 5, "w20", "w32", "w30", [][]byte{[]byte("w30")}, nil},
		play{3, 3, "w23", "w32", "w33", [][]byte{[]byte("w33")}, nil},
		play{1, 7, "w31", "w33", "d13", [][]byte{[]byte("d13")}, nil},
		play{0, 6, "w30", "d13", "w40", [][]byte{[]byte("w40")}, nil},
		play{1, 8, "d13", "w40", "w41", [][]byte{[]byte("w41")}, nil},
		play{2, 7, "w32", "w41", "w42", [][]byte{[]byte("w42")}, nil},
		play{3, 4, "w33", "w42", "w43", [][]byte{[]byte("w43")}, nil},
	}
	if full {
		newPlays := []play{
			play{2, 8, "w42", "w43", "e23", [][]byte{[]byte("e23")}, nil},
			play{1, 9, "w41", "e23", "w51", [][]byte{[]byte("w51")}, nil},
		}
		plays = append(plays, newPlays...)
	}

	playEvents(plays, nodes, index, orderedEvents)

	hashgraph := createHashgraph(false, orderedEvents, participants, logger.WithField("test", 6))

	return hashgraph, index
}

func TestFunkyHashgraphFame(t *testing.T) {
	h, index := initFunkyHashgraph(common.NewTestLogger(t), false)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}

	if l := h.Store.LastRound(); l != 4 {
		t.Fatalf("last round should be 4 not %d", l)
	}

	for r := 0; r < 5; r++ {
		round, err := h.Store.GetRound(r)
		if err != nil {
			t.Fatal(err)
		}
		witnessNames := []string{}
		for _, w := range round.Witnesses() {
			witnessNames = append(witnessNames, getName(index, w))
		}
		t.Logf("Round %d witnesses: %v", r, witnessNames)
	}

	expectedpendingRounds := []pendingRound{
		pendingRound{
			Index:   0,
			Decided: true,
		},
		pendingRound{
			Index:   1,
			Decided: true,
		},
		pendingRound{
			Index:   2,
			Decided: true,
		},
		pendingRound{
			Index:   3,
			Decided: false,
		},
		pendingRound{
			Index:   4,
			Decided: false,
		},
	}

	for i, pd := range h.PendingRounds {
		if !reflect.DeepEqual(*pd, expectedpendingRounds[i]) {
			t.Fatalf("pendingRounds[%d] should be %v, not %v", i, expectedpendingRounds[i], *pd)
		}
	}
}

func TestFunkyHashgraphBlocks(t *testing.T) {
	h, index := initFunkyHashgraph(common.NewTestLogger(t), true)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}
	if err := h.ProcessDecidedRounds(); err != nil {
		t.Fatal(err)
	}

	if l := h.Store.LastRound(); l != 5 {
		t.Fatalf("last round should be 5 not %d", l)
	}

	for r := 0; r < 6; r++ {
		round, err := h.Store.GetRound(r)
		if err != nil {
			t.Fatal(err)
		}
		witnessNames := []string{}
		for _, w := range round.Witnesses() {
			witnessNames = append(witnessNames, getName(index, w))
		}
		t.Logf("Round %d witnesses: %v", r, witnessNames)
	}

	//rounds 0,1, 2 and 3 should be decided
	expectedpendingRounds := []pendingRound{
		pendingRound{
			Index:   4,
			Decided: false,
		},
		pendingRound{
			Index:   5,
			Decided: false,
		},
	}
	for i, pd := range h.PendingRounds {
		if !reflect.DeepEqual(*pd, expectedpendingRounds[i]) {
			t.Fatalf("pendingRounds[%d] should be %v, not %v", i, expectedpendingRounds[i], *pd)
		}
	}

	expectedBlockTxCounts := map[int]int{
		0: 6,
		1: 7,
		2: 7,
	}

	for bi := 0; bi < 3; bi++ {
		b, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}
		for i, tx := range b.Transactions() {
			t.Logf("block %d, tx %d: %s", bi, i, string(tx))
		}
		if txs := len(b.Transactions()); txs != expectedBlockTxCounts[bi] {
			t.Fatalf("Blocks[%d] should contain %d transactions, not %d", bi,
				expectedBlockTxCounts[bi], txs)
		}
	}
}

func TestFunkyHashgraphFrames(t *testing.T) {
	h, index := initFunkyHashgraph(common.NewTestLogger(t), true)

	peerSet, err := h.Store.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}
	if err := h.ProcessDecidedRounds(); err != nil {
		t.Fatal(err)
	}

	t.Logf("------------------------------------------------------------------")
	for bi := 0; bi < 3; bi++ {
		block, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := h.GetFrame(block.RoundReceived())
		for k, ev := range frame.Events {
			r, _ := h.round(ev.Hex())
			t.Logf("frame[%d].Events[%d]: %s, round %d", frame.Round, k, getName(index, ev.Hex()), r)
		}
		for k, r := range frame.Roots {
			t.Logf("frame[%d].Roots[%v]: SelfParent: %v, Others: %v",
				frame.Round, k, r.SelfParent, r.Others)
		}
	}
	t.Logf("------------------------------------------------------------------")

	expectedFrameRoots := map[int]map[string]*Root{
		1: map[string]*Root{
			peerSet.PubKeys()[0]: NewBaseRoot(peerSet.IDs()[0]),
			peerSet.PubKeys()[1]: NewBaseRoot(peerSet.IDs()[1]),
			peerSet.PubKeys()[2]: NewBaseRoot(peerSet.IDs()[2]),
			peerSet.PubKeys()[3]: NewBaseRoot(peerSet.IDs()[3]),
		},
		2: map[string]*Root{
			peerSet.PubKeys()[0]: NewBaseRoot(peerSet.IDs()[0]),
			peerSet.PubKeys()[1]: &Root{
				NextRound:  0,
				SelfParent: RootEvent{index["a12"], peerSet.IDs()[1], 1, 2, 0},
				Others: map[string]RootEvent{
					index["a10"]: RootEvent{index["a00"], peerSet.IDs()[0], 1, 1, 0},
				},
			},
			peerSet.PubKeys()[2]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["a21"], peerSet.IDs()[2], 2, 3, 0},
				Others: map[string]RootEvent{
					index["w12"]: RootEvent{index["w13"], peerSet.IDs()[3], 1, 4, 1},
				},
			},
			peerSet.PubKeys()[3]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["w03"], peerSet.IDs()[3], 0, 0, 0},
				Others: map[string]RootEvent{
					index["w13"]: RootEvent{index["a21"], peerSet.IDs()[2], 2, 3, 0},
				},
			},
		},
		3: map[string]*Root{
			peerSet.PubKeys()[0]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["a00"], peerSet.IDs()[0], 1, 1, 0},
				Others: map[string]RootEvent{
					index["w10"]: RootEvent{index["w11"], peerSet.IDs()[1], 3, 6, 1},
				},
			},
			peerSet.PubKeys()[1]: &Root{
				NextRound:  2,
				SelfParent: RootEvent{index["w11"], peerSet.IDs()[1], 3, 6, 1},
				Others: map[string]RootEvent{
					index["w21"]: RootEvent{index["w23"], peerSet.IDs()[3], 2, 8, 2},
				},
			},
			peerSet.PubKeys()[2]: &Root{
				NextRound:  2,
				SelfParent: RootEvent{index["b21"], peerSet.IDs()[2], 4, 7, 1},
				Others: map[string]RootEvent{
					index["w22"]: RootEvent{index["c10"], peerSet.IDs()[1], 5, 10, 2},
				},
			},
			peerSet.PubKeys()[3]: &Root{
				NextRound:  2,
				SelfParent: RootEvent{index["w13"], peerSet.IDs()[3], 1, 4, 1},
				Others: map[string]RootEvent{
					index["w23"]: RootEvent{index["b21"], peerSet.IDs()[2], 4, 7, 1},
				},
			},
		},
	}

	for bi := 0; bi < 3; bi++ {
		block, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := h.GetFrame(block.RoundReceived())
		if err != nil {
			t.Fatal(err)
		}

		for k, r := range frame.Roots {
			if !reflect.DeepEqual(expectedFrameRoots[frame.Round][k], r) {
				t.Fatalf("frame[%d].Roots[%v] should be %v, not %v", frame.Round, k, expectedFrameRoots[frame.Round][k], r)
			}
		}
	}
}

func TestFunkyHashgraphReset(t *testing.T) {
	h, index := initFunkyHashgraph(common.NewTestLogger(t), true)

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	h.ProcessDecidedRounds()

	for bi := 0; bi < 3; bi++ {
		t.Logf("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
		t.Logf("RESETTING FROM BLOCK %d", bi)
		t.Logf("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")

		block, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := h.GetFrame(block.RoundReceived())
		if err != nil {
			t.Fatal(err)
		}

		//This operation clears the private fields which need to be recomputed
		//in the Events (round, roundReceived,etc)
		marshalledFrame, _ := frame.Marshal()
		unmarshalledFrame := new(Frame)
		unmarshalledFrame.Unmarshal(marshalledFrame)

		peerSet := peers.NewPeerSet(frame.Peers)

		h2 := NewHashgraph(peerSet,
			NewInmemStore(peerSet, cacheSize),
			DummyInternalCommitCallback,
			testLogger(t))
		err = h2.Reset(block, unmarshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		/***********************************************************************
		Test continue after Reset
		***********************************************************************/

		//Compute diff
		h2Known := h2.Store.KnownEvents()
		diff := getDiff(h, h2Known, t)

		wireDiff := make([]WireEvent, len(diff), len(diff))
		for i, e := range diff {
			wireDiff[i] = e.ToWire()
		}

		//Insert remaining Events into the Reset hashgraph
		for i, wev := range wireDiff {
			ev, err := h2.ReadWireInfo(wev)
			if err != nil {
				t.Fatalf("Reading WireInfo for %s: %s", getName(index, diff[i].Hex()), err)
			}
			err = h2.InsertEvent(ev, false)
			if err != nil {
				t.Fatal(err)
			}
		}

		t.Logf("RUN CONSENSUS METHODS*****************************************")
		h2.DivideRounds()
		h2.DecideFame()
		h2.DecideRoundReceived()
		h2.ProcessDecidedRounds()
		t.Logf("**************************************************************")

		compareRoundWitnesses(h, h2, index, bi, true, t)
	}

}

/*


ATTENTION: Look at roots in Rounds 1 and 2

    |   w51   |    |
  ----------- \  ---------------------
	|    |      \  |  Round 4
    |    |    |   i32
    |    |    | /  |
    |    |   w42   |
    |    |  / |    |
    |   w41   |    |
	|    |   \     |
	|    |    | \  |
    |    |    |   w43
--------------------------------------
	|    |    | /  |  Round 3
    |    |   h21   |
    |    | /  |    |
    |   w31   |    |
	|    |   \     |
	|    |    | \  |
    |    |    |   w33
    |    |    | /  |
    |    |   w32   |
--------------------------------------
	|    | /  |    |  Round 2
    |   g13   |    |  ConsensusRound 3
	|    |   \     |
	|    |    | \  |  Frame {
    |    |    |   w23 	Evs  : w21, w22, w23, g13
    |    |    | /  |	Roots: [w00, e32], [w11, w13], [w12, w21], [w13, w22]
    |    |   w22   |  }
    |    | /  |    |
    |   w21   |    |
 ----------- \  ---------------------
	|    |      \  |  Round 1
    |    |    |   w13 ConsensusRound 2
    |    |    | /  |
    |    |   w12   |  Frame {
    |     /   |    |	Evs  : w10, w11, f01, w12, w13
	|  / |    |    |	Roots: [w00,e32], [w10, e10], [w11, e21], [w12, e32]
   f01   |    |    |  }
	| \  |    |    |
    |   w11   |    |
    | /  |    |    |
   w10   |    |    |
-------------------------------------
    |    \    |    |  Round 0
    |    |    \    |  ConsensusRound 1
    |    |    |   e32
    |    |    | /  |  Frame {
    |    |   e21   |     Evs  : w00, w01, w02, w03, e10, e21, e32
    |    | /  |    |     Roots: R0, R1, R2, R3
    |   e10   |    |  }
    |  / |    |    |
   w00  w01  w02  w03
	|    |    |    |
    R0   R1   R2   R3
	0	 1	  2	   3
*/

func initSparseHashgraph(logger *logrus.Logger) (*Hashgraph, map[string]string) {
	nodes, index, orderedEvents, participants := initHashgraphNodes(4)

	for i, peer := range participants.Peers {
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, nil, []string{rootSelfParent(peer.ID), ""}, nodes[i].Pub, 0)
		nodes[i].signAndAddEvent(event, name, index, orderedEvents)
	}

	plays := []play{
		play{1, 1, "w01", "w00", "e10", [][]byte{[]byte("e10")}, nil},
		play{2, 1, "w02", "e10", "e21", [][]byte{[]byte("e21")}, nil},
		play{3, 1, "w03", "e21", "e32", [][]byte{[]byte("e32")}, nil},
		play{0, 1, "w00", "e32", "w10", [][]byte{[]byte("w10")}, nil},
		play{1, 2, "e10", "w10", "w11", [][]byte{[]byte("w11")}, nil},
		play{0, 2, "w10", "w11", "f01", [][]byte{[]byte("f01")}, nil},
		play{2, 2, "e21", "f01", "w12", [][]byte{[]byte("w12")}, nil},
		play{3, 2, "e32", "w12", "w13", [][]byte{[]byte("w13")}, nil},
		play{1, 3, "w11", "w13", "w21", [][]byte{[]byte("w21")}, nil},
		play{2, 3, "w12", "w21", "w22", [][]byte{[]byte("w22")}, nil},
		play{3, 3, "w13", "w22", "w23", [][]byte{[]byte("w23")}, nil},
		play{1, 4, "w21", "w23", "g13", [][]byte{[]byte("g13")}, nil},
		play{2, 4, "w22", "g13", "w32", [][]byte{[]byte("w32")}, nil},
		play{3, 4, "w23", "w32", "w33", [][]byte{[]byte("w33")}, nil},
		play{1, 5, "g13", "w33", "w31", [][]byte{[]byte("w31")}, nil},
		play{2, 5, "w32", "w31", "h21", [][]byte{[]byte("h21")}, nil},
		play{3, 5, "w33", "h21", "w43", [][]byte{[]byte("w43")}, nil},
		play{1, 6, "w31", "w43", "w41", [][]byte{[]byte("w41")}, nil},
		play{2, 6, "h21", "w41", "w42", [][]byte{[]byte("w42")}, nil},
		play{3, 6, "w43", "w42", "i32", [][]byte{[]byte("i32")}, nil},
		play{1, 7, "w41", "i32", "w51", [][]byte{[]byte("w51")}, nil},
	}

	playEvents(plays, nodes, index, orderedEvents)

	hashgraph := createHashgraph(false, orderedEvents, participants, logger.WithField("test", 6))

	return hashgraph, index
}

func TestSparseHashgraphFrames(t *testing.T) {
	h, index := initSparseHashgraph(common.NewTestLogger(t))

	peerSet, err := h.Store.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideFame(); err != nil {
		t.Fatal(err)
	}
	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}
	if err := h.ProcessDecidedRounds(); err != nil {
		t.Fatal(err)
	}

	t.Logf("------------------------------------------------------------------")
	for bi := 0; bi < 3; bi++ {
		block, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := h.GetFrame(block.RoundReceived())
		for k, ev := range frame.Events {
			r, _ := h.round(ev.Hex())
			t.Logf("frame[%d].Events[%d]: %s, round %d", frame.Round, k, getName(index, ev.Hex()), r)
		}
		for k, r := range frame.Roots {
			t.Logf("frame[%d].Roots[%v]: SelfParent: %v, Others: %v",
				frame.Round, k, r.SelfParent, r.Others)
		}
	}
	t.Logf("------------------------------------------------------------------")

	expectedFrameRoots := map[int]map[string]*Root{
		1: map[string]*Root{
			peerSet.PubKeys()[0]: NewBaseRoot(peerSet.IDs()[0]),
			peerSet.PubKeys()[1]: NewBaseRoot(peerSet.IDs()[1]),
			peerSet.PubKeys()[2]: NewBaseRoot(peerSet.IDs()[2]),
			peerSet.PubKeys()[3]: NewBaseRoot(peerSet.IDs()[3]),
		},
		2: map[string]*Root{
			peerSet.PubKeys()[0]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["w00"], peerSet.IDs()[0], 0, 0, 0},
				Others: map[string]RootEvent{
					index["w10"]: RootEvent{index["e32"], peerSet.IDs()[3], 1, 3, 0},
				},
			},
			peerSet.PubKeys()[1]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["e10"], peerSet.IDs()[1], 1, 1, 0},
				Others: map[string]RootEvent{
					index["w11"]: RootEvent{index["w10"], peerSet.IDs()[0], 1, 4, 1},
				},
			},
			peerSet.PubKeys()[2]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["e21"], peerSet.IDs()[2], 1, 2, 0},
				Others: map[string]RootEvent{
					index["w12"]: RootEvent{index["f01"], peerSet.IDs()[0], 2, 6, 1},
				},
			},
			peerSet.PubKeys()[3]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["e32"], peerSet.IDs()[3], 1, 3, 0},
				Others: map[string]RootEvent{
					index["w13"]: RootEvent{index["w12"], peerSet.IDs()[2], 2, 7, 1},
				},
			},
		},
		3: map[string]*Root{
			peerSet.PubKeys()[0]: &Root{
				NextRound:  1,
				SelfParent: RootEvent{index["w10"], peerSet.IDs()[0], 1, 4, 1},
				Others: map[string]RootEvent{
					index["f01"]: RootEvent{index["w11"], peerSet.IDs()[1], 2, 5, 1},
				},
			},
			peerSet.PubKeys()[1]: &Root{
				NextRound:  2,
				SelfParent: RootEvent{index["w11"], peerSet.IDs()[1], 2, 5, 1},
				Others: map[string]RootEvent{
					index["w21"]: RootEvent{index["w13"], peerSet.IDs()[3], 2, 8, 1},
				},
			},
			peerSet.PubKeys()[2]: &Root{
				NextRound:  2,
				SelfParent: RootEvent{index["w12"], peerSet.IDs()[2], 2, 7, 1},
				Others: map[string]RootEvent{
					index["w22"]: RootEvent{index["w21"], peerSet.IDs()[1], 3, 9, 2},
				},
			},
			peerSet.PubKeys()[3]: &Root{
				NextRound:  2,
				SelfParent: RootEvent{index["w13"], peerSet.IDs()[3], 2, 8, 1},
				Others: map[string]RootEvent{
					index["w23"]: RootEvent{index["w22"], peerSet.IDs()[2], 3, 10, 2},
				},
			},
		},
	}

	for bi := 0; bi < 3; bi++ {
		block, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := h.GetFrame(block.RoundReceived())
		if err != nil {
			t.Fatal(err)
		}

		for k, r := range frame.Roots {
			if !reflect.DeepEqual(expectedFrameRoots[frame.Round][k], r) {
				t.Fatalf("frame[%d].Roots[%v] should be %v, not %v", frame.Round, k, expectedFrameRoots[frame.Round][k], r)
			}
		}
	}
}

func TestSparseHashgraphReset(t *testing.T) {
	h, index := initSparseHashgraph(common.NewTestLogger(t))

	peerSet, err := h.Store.GetLastPeerSet()
	if err != nil {
		t.Fatal(err)
	}

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	h.ProcessDecidedRounds()

	for bi := 0; bi < 3; bi++ {
		t.Logf("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
		t.Logf("RESETTING FROM BLOCK %d", bi)
		t.Logf("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")

		block, err := h.Store.GetBlock(bi)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := h.GetFrame(block.RoundReceived())
		if err != nil {
			t.Fatal(err)
		}

		//This operation clears the private fields which need to be recomputed
		//in the Events (round, roundReceived,etc)
		marshalledFrame, _ := frame.Marshal()
		unmarshalledFrame := new(Frame)
		unmarshalledFrame.Unmarshal(marshalledFrame)

		h2 := NewHashgraph(peerSet,
			NewInmemStore(peerSet, cacheSize),
			DummyInternalCommitCallback,
			testLogger(t))
		err = h2.Reset(block, unmarshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		/***********************************************************************
		Test continue after Reset
		***********************************************************************/

		//Compute diff
		h2Known := h2.Store.KnownEvents()
		diff := getDiff(h, h2Known, t)

		t.Logf("h2.Known: %v", h2Known)
		t.Logf("diff: %v", len(diff))

		wireDiff := make([]WireEvent, len(diff), len(diff))
		for i, e := range diff {
			wireDiff[i] = e.ToWire()
		}

		//Insert remaining Events into the Reset hashgraph
		for i, wev := range wireDiff {
			eventName := getName(index, diff[i].Hex())
			ev, err := h2.ReadWireInfo(wev)
			if err != nil {
				t.Fatalf("ReadWireInfo(%s): %s", eventName, err)
			}
			if !reflect.DeepEqual(ev.Body, diff[i].Body) {
				t.Fatalf("%s from WireInfo should be %#v, not %#v", eventName, diff[i].Body, ev.Body)
			}
			err = h2.InsertEvent(ev, false)
			if err != nil {
				t.Fatalf("InsertEvent(%s): %s", eventName, err)
			}
		}

		t.Logf("RUN CONSENSUS METHODS*****************************************")
		h2.DivideRounds()
		h2.DecideFame()
		h2.DecideRoundReceived()
		h2.ProcessDecidedRounds()
		t.Logf("**************************************************************")

		compareRoundWitnesses(h, h2, index, bi, true, t)
	}

}

/*----------------------------------------------------------------------------*/

func compareRoundWitnesses(h, h2 *Hashgraph, index map[string]string, round int, check bool, t *testing.T) {

	for i := round; i <= 5; i++ {
		hRound, err := h.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}
		h2Round, err := h2.Store.GetRound(i)
		if err != nil {
			t.Fatal(err)
		}

		//Check Round1 Witnesses
		hWitnesses := hRound.Witnesses()
		h2Witnesses := h2Round.Witnesses()
		sort.Strings(hWitnesses)
		sort.Strings(h2Witnesses)
		hwn := make([]string, len(hWitnesses))
		h2wn := make([]string, len(h2Witnesses))
		for _, w := range hWitnesses {
			hwn = append(hwn, getName(index, w))
		}
		for _, w := range h2Witnesses {
			h2wn = append(h2wn, getName(index, w))
		}

		t.Logf("h Round%d witnesses: %v", i, hwn)
		t.Logf("h2 Round%d witnesses: %v", i, h2wn)

		if check && !reflect.DeepEqual(hwn, h2wn) {
			t.Fatalf("Reset Hg Round %d witnesses should be %v, not %v", i, hwn, h2wn)
		}
	}

}

func getDiff(h *Hashgraph, known map[int]int, t *testing.T) []*Event {
	peerSet, _ := h.Store.GetLastPeerSet()
	diff := []*Event{}
	for id, ct := range known {
		pk := peerSet.ByID[id].PubKeyHex
		//get participant Events with index > ct
		participantEvents, err := h.Store.ParticipantEvents(pk, ct)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range participantEvents {
			ev, err := h.Store.GetEvent(e)
			if err != nil {
				t.Fatal(err)
			}
			diff = append(diff, ev)
		}
	}
	sort.Sort(ByTopologicalOrder(diff))
	return diff
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

func create(x int) *int {
	return &x
}
