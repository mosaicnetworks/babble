package hashgraph

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/sirupsen/logrus"

	"strings"

	"reflect"

	"math"

	"github.com/mosaicnetworks/babble/common"
	"github.com/mosaicnetworks/babble/crypto"
)

var (
	cacheSize = 100
	n         = 3
	badgerDir = "test_data/badger"
)

type Node struct {
	ID     int
	Pub    []byte
	PubHex string
	Key    *ecdsa.PrivateKey
	Events []Event
}

func NewNode(key *ecdsa.PrivateKey, id int) Node {
	pub := crypto.FromECDSAPub(&key.PublicKey)
	node := Node{
		ID:     id,
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
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key, i)
		event := NewEvent(nil, nil, []string{"", ""}, node.Pub, 0)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

	plays := []play{
		play{0, 1, "e0", "e1", "e01", nil, nil},
		play{2, 1, "e2", "", "s20", nil, nil},
		play{1, 1, "e1", "", "s10", nil, nil},
		play{0, 2, "e01", "", "s00", nil, nil},
		play{2, 2, "s20", "s00", "e20", nil, nil},
		play{1, 2, "s10", "e20", "e12", nil, nil},
	}

	for _, p := range plays {
		e := NewEvent(p.txPayload,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		nodes[p.to].signAndAddEvent(e, p.name, index, orderedEvents)
	}

	participants := make(map[string]int)
	for _, node := range nodes {
		participants[node.PubHex] = node.ID
	}

	store := NewInmemStore(participants, cacheSize)
	h := NewHashgraph(participants, store, nil, testLogger(t))
	for i, ev := range *orderedEvents {
		if err := h.initEventCoordinates(&ev); err != nil {
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

	//1 generation
	if !h.ancestor(index["e01"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e01")
	}
	if !h.ancestor(index["e01"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e01")
	}
	if !h.ancestor(index["s00"], index["e01"]) {
		t.Fatal("e01 should be ancestor of s00")
	}
	if !h.ancestor(index["s20"], index["e2"]) {
		t.Fatal("e2 should be ancestor of s20")
	}
	if !h.ancestor(index["e20"], index["s00"]) {
		t.Fatal("s00 should be ancestor of e20")
	}
	if !h.ancestor(index["e20"], index["s20"]) {
		t.Fatal("s20 should be ancestor of e20")
	}
	if !h.ancestor(index["e12"], index["e20"]) {
		t.Fatal("e20 should be ancestor of e12")
	}
	if !h.ancestor(index["e12"], index["s10"]) {
		t.Fatal("s10 should be ancestor of e12")
	}

	//2 generations
	if !h.ancestor(index["s00"], index["e0"]) {
		t.Fatalf("e0 should be ancestor of s00")
	}
	if !h.ancestor(index["s00"], index["e1"]) {
		t.Fatalf("e1 should be ancestor of s00")
	}
	if !h.ancestor(index["e20"], index["e01"]) {
		t.Fatalf("e01 should be ancestor of e20")
	}
	if !h.ancestor(index["e20"], index["e2"]) {
		t.Fatalf("e2 should be ancestor of e20")
	}
	if !h.ancestor(index["e12"], index["e1"]) {
		t.Fatalf("e1 should be ancestor of e12")
	}
	if !h.ancestor(index["e12"], index["s20"]) {
		t.Fatalf("s20 should be ancestor of e12")
	}

	//3 generations
	if !h.ancestor(index["e20"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e20")
	}
	if !h.ancestor(index["e20"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e20")
	}
	if !h.ancestor(index["e20"], index["e2"]) {
		t.Fatal("e2 should be ancestor of e20")
	}
	if !h.ancestor(index["e12"], index["e01"]) {
		t.Fatal("e01 should be ancestor of e12")
	}
	if !h.ancestor(index["e12"], index["e0"]) {
		t.Fatal("e0 should be ancestor of e12")
	}
	if !h.ancestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be ancestor of e12")
	}
	if !h.ancestor(index["e12"], index["e2"]) {
		t.Fatal("e2 should be ancestor of e12")
	}

	//false positive
	if h.ancestor(index["e01"], index["e2"]) {
		t.Fatal("e2 should not be ancestor of e01")
	}
	if h.ancestor(index["s00"], index["e2"]) {
		t.Fatal("e2 should not be ancestor of s00")
	}

	if h.ancestor(index["e0"], "") {
		t.Fatal("\"\" should not be ancestor of e0")
	}
	if h.ancestor(index["s00"], "") {
		t.Fatal("\"\" should not be ancestor of s00")
	}
	if h.ancestor(index["e12"], "") {
		t.Fatal("\"\" should not be ancestor of e12")
	}

}

func TestSelfAncestor(t *testing.T) {
	h, index := initHashgraph(t)

	//1 generation
	if !h.selfAncestor(index["e01"], index["e0"]) {
		t.Fatal("e0 should be self ancestor of e01")
	}
	if !h.selfAncestor(index["s00"], index["e01"]) {
		t.Fatal("e01 should be self ancestor of s00")
	}

	//1 generation false negatives
	if h.selfAncestor(index["e01"], index["e1"]) {
		t.Fatal("e1 should not be self ancestor of e01")
	}
	if h.selfAncestor(index["e12"], index["e20"]) {
		t.Fatal("e20 should not be self ancestor of e12")
	}
	if h.selfAncestor(index["s20"], "") {
		t.Fatal("\"\" should not be self ancestor of s20")
	}

	//2 generation
	if !h.selfAncestor(index["e20"], index["e2"]) {
		t.Fatal("e2 should be self ancestor of e20")
	}
	if !h.selfAncestor(index["e12"], index["e1"]) {
		t.Fatal("e1 should be self ancestor of e12")
	}

	//2 generation false negative
	if h.selfAncestor(index["e20"], index["e0"]) {
		t.Fatal("e0 should not be self ancestor of e20")
	}
	if h.selfAncestor(index["e12"], index["e2"]) {
		t.Fatal("e2 should not be self ancestor of e12")
	}
	if h.selfAncestor(index["e20"], index["e01"]) {
		t.Fatal("e01 should not be self ancestor of e20")
	}

}

func TestSee(t *testing.T) {
	h, index := initHashgraph(t)

	if !h.see(index["e01"], index["e0"]) {
		t.Fatal("e01 should see e0")
	}
	if !h.see(index["e01"], index["e1"]) {
		t.Fatal("e01 should see e1")
	}
	if !h.see(index["e20"], index["e0"]) {
		t.Fatal("e20 should see e0")
	}
	if !h.see(index["e20"], index["e01"]) {
		t.Fatal("e20 should see e01")
	}
	if !h.see(index["e12"], index["e01"]) {
		t.Fatal("e12 should see e01")
	}
	if !h.see(index["e12"], index["e0"]) {
		t.Fatal("e12 should see e0")
	}
	if !h.see(index["e12"], index["e1"]) {
		t.Fatal("e12 should see e1")
	}
	if !h.see(index["e12"], index["s20"]) {
		t.Fatal("e12 should see s20")
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

	for e, et := range expectedTimestamps {
		t.Logf("%s: %d", e, h.lamportTimestamp(index[e]))
		if ts := h.lamportTimestamp(index[e]); ts != et {
			t.Fatalf("%s LamportTimestamp should be %d, not %d", e, et, ts)
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
	nodes := []Node{}
	participants := make(map[string]int)

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key, i)
		nodes = append(nodes, node)
		participants[node.PubHex] = node.ID
	}

	store := NewInmemStore(participants, cacheSize)
	hashgraph := NewHashgraph(participants, store, nil, testLogger(t))

	for i, node := range nodes {
		event := NewEvent(nil, nil, []string{"", ""}, node.Pub, 0)
		event.Sign(node.Key)
		index[fmt.Sprintf("e%d", i)] = event.Hex()
		hashgraph.InsertEvent(event, true)
	}

	//a and e2 need to have different hashes
	eventA := NewEvent([][]byte{[]byte("yo")}, nil, []string{"", ""}, nodes[2].Pub, 0)
	eventA.Sign(nodes[2].Key)
	index["a"] = eventA.Hex()
	if err := hashgraph.InsertEvent(eventA, true); err == nil {
		t.Fatal("InsertEvent should return error for 'a'")
	}

	event01 := NewEvent(nil, nil,
		[]string{index["e0"], index["a"]}, //e0 and a
		nodes[0].Pub, 1)
	event01.Sign(nodes[0].Key)
	index["e01"] = event01.Hex()
	if err := hashgraph.InsertEvent(event01, true); err == nil {
		t.Fatal("InsertEvent should return error for e01")
	}

	event20 := NewEvent(nil, nil,
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
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key, i)
		event := NewEvent(nil, nil, []string{"", ""}, node.Pub, 0)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

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

	for _, p := range plays {
		e := NewEvent(p.txPayload,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		nodes[p.to].signAndAddEvent(e, p.name, index, orderedEvents)
	}

	participants := make(map[string]int)
	for _, node := range nodes {
		participants[node.PubHex] = node.ID
	}

	hashgraph := NewHashgraph(participants, NewInmemStore(participants, cacheSize), nil, testLogger(t))
	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev, true); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}
	return hashgraph, index
}

func TestInsertEvent(t *testing.T) {
	h, index := initRoundHashgraph(t)

	t.Run("Check Event Coordinates", func(t *testing.T) {

		expectedFirstDescendants := make([]EventCoordinates, n)
		expectedLastAncestors := make([]EventCoordinates, n)

		//e0
		e0, err := h.Store.GetEvent(index["e0"])
		if err != nil {
			t.Fatal(err)
		}

		if !(e0.Body.selfParentIndex == -1 &&
			e0.Body.otherParentCreatorID == -1 &&
			e0.Body.otherParentIndex == -1 &&
			e0.Body.creatorID == h.Participants[e0.Creator()]) {
			t.Fatalf("Invalid wire info on e0")
		}

		expectedFirstDescendants[0] = EventCoordinates{
			index: 0,
			hash:  index["e0"],
		}
		expectedFirstDescendants[1] = EventCoordinates{
			index: 1,
			hash:  index["e10"],
		}
		expectedFirstDescendants[2] = EventCoordinates{
			index: 2,
			hash:  index["e21"],
		}

		expectedLastAncestors[0] = EventCoordinates{
			index: 0,
			hash:  index["e0"],
		}
		expectedLastAncestors[1] = EventCoordinates{
			index: -1,
		}
		expectedLastAncestors[2] = EventCoordinates{
			index: -1,
		}

		if !reflect.DeepEqual(e0.firstDescendants, expectedFirstDescendants) {
			t.Fatal("e0 firstDescendants not good")
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
			e21.Body.otherParentCreatorID == h.Participants[e10.Creator()] &&
			e21.Body.otherParentIndex == 1 &&
			e21.Body.creatorID == h.Participants[e21.Creator()]) {
			t.Fatalf("Invalid wire info on e21")
		}

		expectedFirstDescendants[0] = EventCoordinates{
			index: 2,
			hash:  index["e02"],
		}
		expectedFirstDescendants[1] = EventCoordinates{
			index: 3,
			hash:  index["f1"],
		}
		expectedFirstDescendants[2] = EventCoordinates{
			index: 2,
			hash:  index["e21"],
		}

		expectedLastAncestors[0] = EventCoordinates{
			index: 0,
			hash:  index["e0"],
		}
		expectedLastAncestors[1] = EventCoordinates{
			index: 1,
			hash:  index["e10"],
		}
		expectedLastAncestors[2] = EventCoordinates{
			index: 2,
			hash:  index["e21"],
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
			f1.Body.otherParentCreatorID == h.Participants[e0.Creator()] &&
			f1.Body.otherParentIndex == 2 &&
			f1.Body.creatorID == h.Participants[f1.Creator()]) {
			t.Fatalf("Invalid wire info on f1")
		}

		expectedFirstDescendants[0] = EventCoordinates{
			index: math.MaxInt32,
		}
		expectedFirstDescendants[1] = EventCoordinates{
			index: 3,
			hash:  index["f1"],
		}
		expectedFirstDescendants[2] = EventCoordinates{
			index: math.MaxInt32,
		}

		expectedLastAncestors[0] = EventCoordinates{
			index: 2,
			hash:  index["e02"],
		}
		expectedLastAncestors[1] = EventCoordinates{
			index: 3,
			hash:  index["f1"],
		}
		expectedLastAncestors[2] = EventCoordinates{
			index: 2,
			hash:  index["e21"],
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

	if !h.stronglySee(index["e21"], index["e0"]) {
		t.Fatal("e21 should strongly see e0")
	}

	if !h.stronglySee(index["e02"], index["e10"]) {
		t.Fatal("e02 should strongly see e10")
	}
	if !h.stronglySee(index["e02"], index["e0"]) {
		t.Fatal("e02 should strongly see e0")
	}
	if !h.stronglySee(index["e02"], index["e1"]) {
		t.Fatal("e02 should strongly see e1")
	}

	if !h.stronglySee(index["f1"], index["e21"]) {
		t.Fatal("f1 should strongly see e21")
	}
	if !h.stronglySee(index["f1"], index["e10"]) {
		t.Fatal("f1 should strongly see e10")
	}
	if !h.stronglySee(index["f1"], index["e0"]) {
		t.Fatal("f1 should strongly see e0")
	}
	if !h.stronglySee(index["f1"], index["e1"]) {
		t.Fatal("f1 should strongly see e1")
	}
	if !h.stronglySee(index["f1"], index["e2"]) {
		t.Fatal("f1 should strongly see e2")
	}
	if !h.stronglySee(index["s11"], index["e2"]) {
		t.Fatal("s11 should strongly see e2")
	}

	//false negatives
	if h.stronglySee(index["e10"], index["e0"]) {
		t.Fatal("e12 should not strongly see e2")
	}
	if h.stronglySee(index["e21"], index["e1"]) {
		t.Fatal("e21 should not strongly see e1")
	}
	if h.stronglySee(index["e21"], index["e2"]) {
		t.Fatal("e21 should not strongly see e2")
	}
	if h.stronglySee(index["e02"], index["e2"]) {
		t.Fatal("e02 should not strongly see e2")
	}
	if h.stronglySee(index["s11"], index["e02"]) {
		t.Fatal("s11 should not strongly see e02")
	}
}

func TestParentRound(t *testing.T) {
	h, index := initRoundHashgraph(t)

	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e1"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, RoundInfo{Events: round0Witnesses})

	round1Witnesses := make(map[string]RoundEvent)
	round1Witnesses[index["f1"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(1, RoundInfo{Events: round1Witnesses})

	if r := h.parentRound(index["e0"]).round; r != -1 {
		t.Fatalf("e0.ParentRound().round should be -1, not %d", r)
	}
	if r := h.parentRound(index["e0"]).isRoot; !r {
		t.Fatal("e0.ParentRound().isRoot should be true")
	}

	if r := h.parentRound(index["e1"]).round; r != -1 {
		t.Fatalf("e1.ParentRound().round should be -1, not %d", r)
	}
	if r := h.parentRound(index["e1"]).isRoot; !r {
		t.Fatal("e1.ParentRound().isRoot should be true")
	}

	if r := h.parentRound(index["f1"]).round; r != 0 {
		t.Fatalf("f1.ParentRound().round should be 0, not %d", r)
	}
	if r := h.parentRound(index["f1"]).isRoot; r {
		t.Fatalf("f1.ParentRound().isRoot should be false")
	}

	if r := h.parentRound(index["s11"]).round; r != 1 {
		t.Fatalf("s11.ParentRound().round should be 1, not %d", r)
	}
	if r := h.parentRound(index["s11"]).isRoot; r {
		t.Fatalf("s11.ParentRound().isRoot should be false")
	}
}

func TestWitness(t *testing.T) {
	h, index := initRoundHashgraph(t)

	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e1"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, RoundInfo{Events: round0Witnesses})

	round1Witnesses := make(map[string]RoundEvent)
	round1Witnesses[index["f1"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(1, RoundInfo{Events: round1Witnesses})

	if !h.witness(index["e0"]) {
		t.Fatalf("e0 should be witness")
	}
	if !h.witness(index["e1"]) {
		t.Fatalf("e1 should be witness")
	}
	if !h.witness(index["e2"]) {
		t.Fatalf("e2 should be witness")
	}
	if !h.witness(index["f1"]) {
		t.Fatalf("f1 should be witness")
	}

	if h.witness(index["e10"]) {
		t.Fatalf("e10 should not be witness")
	}
	if h.witness(index["e21"]) {
		t.Fatalf("e21 should not be witness")
	}
	if h.witness(index["e02"]) {
		t.Fatalf("e02 should not be witness")
	}
}

func TestRoundInc(t *testing.T) {
	h, index := initRoundHashgraph(t)

	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e1"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, RoundInfo{Events: round0Witnesses})

	if !h.roundInc(index["f1"]) {
		t.Fatal("RoundInc f1 should be true")
	}

	if h.roundInc(index["e02"]) {
		t.Fatal("RoundInc e02 should be false because it doesnt strongly see e2")
	}
}

func TestRound(t *testing.T) {
	h, index := initRoundHashgraph(t)

	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e1"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, RoundInfo{Events: round0Witnesses})

	if r := h.round(index["f1"]); r != 1 {
		t.Fatalf("round of f1 should be 1 not %d", r)
	}
	if r := h.round(index["e02"]); r != 0 {
		t.Fatalf("round of e02 should be 0 not %d", r)
	}

}

func TestRoundDiff(t *testing.T) {
	h, index := initRoundHashgraph(t)

	round0Witnesses := make(map[string]RoundEvent)
	round0Witnesses[index["e0"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e1"]] = RoundEvent{Witness: true, Famous: Undefined}
	round0Witnesses[index["e2"]] = RoundEvent{Witness: true, Famous: Undefined}
	h.Store.SetRound(0, RoundInfo{Events: round0Witnesses})

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

	round0, err := h.Store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(round0.Witnesses()); l != 3 {
		t.Fatalf("round 0 should have 3 witnesses, not %d", l)
	}
	if !contains(round0.Witnesses(), index["e0"]) {
		t.Fatalf("round 0 witnesses should contain e0")
	}
	if !contains(round0.Witnesses(), index["e1"]) {
		t.Fatalf("round 0 witnesses should contain e1")
	}
	if !contains(round0.Witnesses(), index["e2"]) {
		t.Fatalf("round 0 witnesses should contain e2")
	}

	round1, err := h.Store.GetRound(1)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(round1.Witnesses()); l != 1 {
		t.Fatalf("round 1 should have 1 witness, not %d", l)
	}
	if !contains(round1.Witnesses(), index["f1"]) {
		t.Fatalf("round 1 witnesses should contain f1")
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

func contains(s []string, x string) bool {
	for _, e := range s {
		if e == x {
			return true
		}
	}
	return false
}

/*

e0  e1  e2    Block (0, 1)
0   1    2
*/
func initBlockHashgraph(t *testing.T) (*Hashgraph, []Node, map[string]string) {
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	//create the initial events
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key, i)
		event := NewEvent(nil, nil, []string{"", ""}, node.Pub, 0)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

	participants := make(map[string]int)
	for _, node := range nodes {
		participants[node.PubHex] = node.ID
	}

	hashgraph := NewHashgraph(participants, NewInmemStore(participants, cacheSize), nil, testLogger(t))

	//create a block and signatures manually
	block := NewBlock(0, 1, []byte("framehash"), [][]byte{[]byte("block tx")})
	err := hashgraph.Store.SetBlock(block)
	if err != nil {
		t.Fatalf("Error setting block. Err: %s", err)
	}

	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev, true); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
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

		block1 := NewBlock(1, 2, []byte("framehash"), [][]byte{})
		sig, _ := block1.Sign(nodes[2].Key)

		//unknown block
		unknownBlockSig := BlockSignature{
			Validator: nodes[2].Pub,
			Index:     1,
			Signature: sig.Signature,
		}
		p := play{2, 2, "s20", "e10", "e21", nil, []BlockSignature{unknownBlockSig}}

		e := NewEvent(nil,
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
		_, err := h.Store.GetEvent(index["e21"])
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
		badNode := NewNode(key, 666)
		badNodeSig, _ := block.Sign(badNode.Key)

		p := play{0, 2, "s00", "e21", "e02", nil, []BlockSignature{badNodeSig}}

		e := NewEvent(nil,
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
	---f0x  |   f21 //f0x's other-parent is e21b. This situation can happen with concurrency
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
	----------- e21b                +---------+
		|   |   |
		|   |  e21
		|   | / |
		|  e10  |
	    | / |   |
		e0  e1  e2
		0   1    2
*/
func initConsensusHashgraph(db bool, t testing.TB) (*Hashgraph, map[string]string) {
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key, i)
		event := NewEvent(nil, nil, []string{"", ""}, node.Pub, 0)
		node.signAndAddEvent(event, fmt.Sprintf("e%d", i), index, orderedEvents)
		nodes = append(nodes, node)
	}

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
		play{0, 3, "f0", "e21b", "f0x", nil, nil},
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

	for _, p := range plays {
		e := NewEvent(p.txPayload,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		nodes[p.to].signAndAddEvent(e, p.name, index, orderedEvents)
	}

	participants := make(map[string]int)
	for _, node := range nodes {
		participants[node.PubHex] = node.ID
	}

	var store Store
	if db {
		var err error
		store, err = NewBadgerStore(participants, cacheSize, badgerDir)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		store = NewInmemStore(participants, cacheSize)
	}

	hashgraph := NewHashgraph(participants, store, nil, testLogger(t))

	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev, true); err != nil {
			t.Fatalf("ERROR inserting event %d: %s\n", i, err)
		}
	}

	return hashgraph, index
}

func TestDivideRoundsBis(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	if err := h.DivideRounds(); err != nil {
		t.Fatal(err)
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

	round0, err := h.Store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}
	if f := round0.Events[index["e0"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("e0 should be famous; got %v", f)
	}
	if f := round0.Events[index["e1"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("e1 should be famous; got %v", f)
	}
	if f := round0.Events[index["e2"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("e2 should be famous; got %v", f)
	}

	round1, err := h.Store.GetRound(1)
	if err != nil {
		t.Fatal(err)
	}
	if f := round1.Events[index["f0"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("f0 should be famous; got %v", f)
	}
	if f := round1.Events[index["f1"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("f1 should be famous; got %v", f)
	}
	if f := round1.Events[index["f2"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("f2 should be famous; got %v", f)
	}

	round2, err := h.Store.GetRound(2)
	if err != nil {
		t.Fatal(err)
	}
	if f := round2.Events[index["g0"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("g0 should be famous; got %v", f)
	}
	if f := round2.Events[index["g1"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("g1 should be famous; got %v", f)
	}
	if f := round2.Events[index["g2"]]; !(f.Witness && f.Famous == True) {
		t.Fatalf("g2 should be famous; got %v", f)
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
	if ce := len(round1.ConsensusEvents()); ce != 7 {
		t.Fatalf("Round 1 should contain 7 ConsensusEvents, not %d", ce)
	}

	round2, err := h.Store.GetRound(2)
	if err != nil {
		t.Fatalf("Could not retrieve Round 2. %s", err)
	}
	if ce := len(round2.ConsensusEvents()); ce != 9 {
		t.Fatalf("Round 1 should contain 9 ConsensusEvents, not %d", ce)
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

	if v := h.LowestRoundWithUndeterminedEvents; v == nil || *v != 2 {
		t.Fatalf("LowestRoundWithUndeterminedEvents should be 2, not %d", v)
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
	if v := h.LowestRoundWithUndeterminedEvents; v == nil || *v != 2 {
		t.Fatalf("LowestRoundWithUndecidedEvents should be 2, not %d", v)
	}
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

	expectedKnown := map[int]int{
		0: 10,
		1: 9,
		2: 9,
	}

	known := h.Store.KnownEvents()
	for _, id := range h.Participants {
		if l := known[id]; l != expectedKnown[id] {
			t.Fatalf("Known[%d] should be %d, not %d", id, expectedKnown[id], l)
		}
	}
}

func TestGetFrame(t *testing.T) {
	h, index := initConsensusHashgraph(false, t)

	h.DivideRounds()
	h.DecideFame()
	h.DecideRoundReceived()
	h.ProcessDecidedRounds()

	t.Run("Round 1", func(t *testing.T) {
		expectedRoots := make([]Root, n)
		expectedRoots[0] = NewBaseRoot()
		expectedRoots[1] = NewBaseRoot()
		expectedRoots[2] = NewBaseRoot()

		frame, err := h.GetFrame(1)
		if err != nil {
			t.Fatal(err)
		}

		for p, r := range frame.Roots {
			er := expectedRoots[p]
			if x := r.X; !reflect.DeepEqual(x, er.X) {
				t.Fatalf("Roots[%d].X should be %v, not %v", p, er.X, x)
			}
			if y := r.Y; !reflect.DeepEqual(y, er.Y) {
				t.Fatalf("Roots[%d].Y should be %v, not %v", p, er.Y, y)
			}
			if others := r.Others; !reflect.DeepEqual(others, er.Others) {
				t.Fatalf("Roots[%d].Others should be %v, not %vv", p, er.Others, others)
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
		expectedEvents := []Event{}
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
		expectedRoots := make([]Root, n)
		expectedRoots[0] = Root{
			X:     RootEvent{index["e02"], 4, 0, 3, true},
			Y:     RootEvent{index["f1b"], 6, 1, 0, true},
			Index: 1,
			Others: map[string]RootEvent{
				index["f0x"]: RootEvent{
					Hash:             index["e21b"],
					LamportTimestamp: 3,
					Round:            0,
					DescendantStronglySeenWitnesses: 3,
					DescendantWitness:               false,
				},
			},
		}
		expectedRoots[1] = Root{
			X:      RootEvent{index["e10"], 1, 0, 3, true},
			Y:      RootEvent{index["e02"], 4, 0, 3, true},
			Index:  1,
			Others: map[string]RootEvent{},
		}
		expectedRoots[2] = Root{
			X:      RootEvent{index["e21b"], 3, 0, 3, true},
			Y:      RootEvent{index["f1b"], 6, 1, 0, true},
			Index:  2,
			Others: map[string]RootEvent{},
		}

		frame, err := h.GetFrame(2)
		if err != nil {
			t.Fatal(err)
		}

		for p, r := range frame.Roots {
			er := expectedRoots[p]
			if x := r.X; !reflect.DeepEqual(x, er.X) {
				t.Fatalf("Roots[%d].X should be %v, not %v", p, er.X, x)
			}
			if y := r.Y; !reflect.DeepEqual(y, er.Y) {
				t.Fatalf("Roots[%d].Y should be %v, not %v", p, er.Y, y)
			}
			if others := r.Others; !reflect.DeepEqual(others, er.Others) {
				t.Fatalf("Roots[%d].Others should be %v, not %v", p, er.Others, others)
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
		expectedEvents := []Event{}
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

	h2 := NewHashgraph(h.Participants,
		NewInmemStore(h.Participants, cacheSize),
		nil,
		testLogger(t))
	err = h2.Reset(block, *unmarshalledFrame)
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
		0: 5,
		1: 4,
		2: 4,
	}

	known := h2.Store.KnownEvents()
	for _, id := range h2.Participants {
		if l := known[id]; l != expectedKnown[id] {
			t.Fatalf("Known[%d] should be %d, not %d", id, expectedKnown[id], l)
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
		if h2r := h2.round(ev.Hex()); h2r != h.round(ev.Hex()) {
			t.Fatalf("h2[%v].Round should be %d, not %d", getName(index, ev.Hex()), h.round(ev.Hex()), h2r)
		}
		if h2s := h2.lamportTimestamp(ev.Hex()); h2s != h.lamportTimestamp(ev.Hex()) {
			t.Fatalf("h2[%v].LamportTimestamp should be %d, not %d", getName(index, ev.Hex()), h.lamportTimestamp(ev.Hex()), h2s)
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

	if v := h2.LowestRoundWithUndeterminedEvents; v == nil || *v != 1 {
		t.Fatalf("LowestRoundWithUndeterminedEvents should be 1, not %v", *v)
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

		events := []Event{}
		for _, e := range round.RoundEvents() {
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

			err = h2.InsertEvent(*unmarshalledEv, true)
			if err != nil {
				t.Fatalf("ERR Inserting Event %s: %v", getName(index, ev.Hex()), err)
			}
		}
	}

	h2.DivideRounds()
	h2.DecideFame()
	h2.DecideRoundReceived()
	h2.ProcessDecidedRounds()

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
	nh := NewHashgraph(recycledStore.participants,
		recycledStore,
		nil,
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

	This example demonstrates that a Round can be 'decided' before an earlier
	round. Here, rounds 1 and 2 are decided before round 0 because the fame of
	witness w00 is only decided at round 5.

--------------------------------------------------------------------------------
	|   w51   |    | This section is only added in 'full' mode
    |    |  \ |    | w51 collects votes from w40, w41, w42, and w43. It DECIDES
	|    |   e23   | yes.
----------------\--------------------------------------------------------------
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
	index := make(map[string]string)
	nodes := []Node{}
	orderedEvents := &[]Event{}

	n := 4
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		node := NewNode(key, i)
		name := fmt.Sprintf("w0%d", i)
		event := NewEvent([][]byte{[]byte(name)}, nil, []string{"", ""}, node.Pub, 0)
		node.signAndAddEvent(event, name, index, orderedEvents)
		nodes = append(nodes, node)
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

	for _, p := range plays {
		e := NewEvent(p.txPayload,
			p.sigPayload,
			[]string{index[p.selfParent], index[p.otherParent]},
			nodes[p.to].Pub,
			p.index)
		nodes[p.to].signAndAddEvent(e, p.name, index, orderedEvents)
	}

	participants := make(map[string]int)
	for _, node := range nodes {
		participants[node.PubHex] = node.ID
	}

	hashgraph := NewHashgraph(participants,
		NewInmemStore(participants, cacheSize),
		nil, logger.WithField("test", 6))

	for i, ev := range *orderedEvents {
		if err := hashgraph.InsertEvent(ev, true); err != nil {
			fmt.Printf("ERROR inserting event %d: %s\n", i, err)
		}
	}

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

	//Rounds 1 and 2 should get decided BEFORE round 0
	expectedpendingRounds := []pendingRound{
		pendingRound{
			Index:   0,
			Decided: false,
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

	if err := h.DecideRoundReceived(); err != nil {
		t.Fatal(err)
	}
	if err := h.ProcessDecidedRounds(); err != nil {
		t.Fatal(err)
	}

	//But a dicided round should never be processed until all previous rounds
	//are decided. So the PendingQueue should remain the same after calling
	//ProcessDecidedRounds()

	for i, pd := range h.PendingRounds {
		if !reflect.DeepEqual(*pd, expectedpendingRounds[i]) {
			t.Fatalf("pendingRounds[%d] should be %v, not %v", i, expectedpendingRounds[i], *pd)
		}
	}

	if v := h.LowestRoundWithUndeterminedEvents; v == nil || *v != 1 {
		t.Fatalf("LowestRoundWithUndecidedEvents should be 1, not %d", v)
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
			t.Logf("frame[%d].Events[%d]: %s, round %d", frame.Round, k, getName(index, ev.Hex()), h.round(ev.Hex()))
		}
		for k, r := range frame.Roots {
			t.Logf("frame[%d].Roots[%d]: X: %v, Y: %v, Others: %v",
				frame.Round, k, r.X, r.Y, r.Others)
		}
	}
	t.Logf("------------------------------------------------------------------")

	expectedFrameRoots := map[int][]Root{
		1: []Root{
			NewBaseRoot(),
			NewBaseRoot(),
			NewBaseRoot(),
			NewBaseRoot(),
		},
		2: []Root{
			NewBaseRoot(),
			Root{
				X:      RootEvent{index["a12"], 2, 0, 1, false},
				Y:      RootEvent{index["a00"], 1, 0, 1, false},
				Index:  1,
				Others: map[string]RootEvent{},
			},
			Root{
				X:      RootEvent{index["a21"], 3, 0, 3, true},
				Y:      RootEvent{index["w13"], 4, 1, 0, true},
				Index:  2,
				Others: map[string]RootEvent{},
			},
			Root{
				X:      RootEvent{index["w03"], 0, 0, 3, true},
				Y:      RootEvent{index["a21"], 3, 0, 3, true},
				Index:  0,
				Others: map[string]RootEvent{},
			},
		},
		3: []Root{
			Root{
				X:      RootEvent{index["a00"], 1, 0, 3, true},
				Y:      RootEvent{index["w11"], 6, 1, 2, true},
				Index:  1,
				Others: map[string]RootEvent{},
			},
			Root{
				X:      RootEvent{index["w11"], 6, 1, 3, true},
				Y:      RootEvent{index["w23"], 8, 2, 0, true},
				Index:  3,
				Others: map[string]RootEvent{},
			},
			Root{
				X:      RootEvent{index["b21"], 7, 1, 4, true},
				Y:      RootEvent{index["c10"], 10, 2, 1, true},
				Index:  4,
				Others: map[string]RootEvent{},
			},
			Root{
				X:      RootEvent{index["w13"], 4, 1, 3, true},
				Y:      RootEvent{index["b21"], 7, 1, 3, true},
				Index:  1,
				Others: map[string]RootEvent{},
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
				t.Fatalf("frame[%d].Roots[%d] should be %v, not %v", frame.Round, k, expectedFrameRoots[frame.Round][k], r)
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

		h2 := NewHashgraph(h.Participants,
			NewInmemStore(h.Participants, cacheSize),
			nil,
			testLogger(t))
		err = h2.Reset(block, *unmarshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		/***********************************************************************
		Test continue after Reset
		***********************************************************************/

		//Compute diff
		h2Known := h2.Store.KnownEvents()
		diff := getDiff(h, h2Known, t)

		//Insert remaining Events into the Reset hashgraph
		for _, ev := range diff {
			marshalledEv, _ := ev.Marshal()
			unmarshalledEv := new(Event)
			unmarshalledEv.Unmarshal(marshalledEv)
			err := h2.InsertEvent(*unmarshalledEv, true)
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

func getDiff(h *Hashgraph, known map[int]int, t *testing.T) []Event {
	diff := []Event{}
	for id, ct := range known {
		pk := h.ReverseParticipants[id]
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
