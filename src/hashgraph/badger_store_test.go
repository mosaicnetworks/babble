package hashgraph

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/src/peers"
)

func initBadgerStore(cacheSize int, t *testing.T) *BadgerStore {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)
	dir, err := ioutil.TempDir("test_data", "badger")
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewBadgerStore(cacheSize, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func removeBadgerStore(store *BadgerStore, t *testing.T) {
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(store.path); err != nil {
		t.Fatal(err)
	}
}

/*******************************************************************************
Test creating, loading, and closing a BadgerStore
*******************************************************************************/

func TestNewBadgerStore(t *testing.T) {
	store := initBadgerStore(1000, t)

	if _, err := os.Stat(store.path); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

/*******************************************************************************
Call DB methods directly
*******************************************************************************/

func TestDBRepertoireMethods(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, _ := initPeers(3)

	for _, p := range peerSet.Peers {
		if err := store.dbSetRepertoire(p); err != nil {
			t.Fatal(err)
		}
	}

	repertoire, err := store.dbGetRepertoire()
	if err != nil {
		t.Fatal(err)
	}

	//force computation of id field, otherwise DeepEquals will fail.
	for _, p := range repertoire {
		p.ID()
	}

	if !reflect.DeepEqual(peerSet.ByPubKey, repertoire) {
		t.Fatalf("Repertoire should be %#v, not %#v", peerSet.ByPubKey, repertoire)
	}

}

func TestDBPeerSetMethods(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, _ := initPeers(3)

	err := store.dbSetPeerSet(0, peerSet)
	if err != nil {
		t.Fatal(err)
	}

	peerSet0, err := store.dbGetPeerSet(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(peerSet, peerSet0) {
		t.Fatalf("Retrieved PeerSet should be %#v, not %#v", peerSet, peerSet0)
	}
}

func TestDBEventMethods(t *testing.T) {
	cacheSize := 0
	testSize := 100

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	_, participants := initPeers(3)

	//insert events in db directly
	events := make(map[string][]*Event)
	topologicalIndex := 0
	topologicalEvents := []*Event{}
	for _, p := range participants {
		items := []*Event{}
		for k := 0; k < testSize; k++ {
			event := NewEvent(
				[][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], k))},
				[]InternalTransaction{},
				[]BlockSignature{{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
				[]string{"", ""},
				p.pubKey,
				k)
			event.Sign(p.privKey)
			event.topologicalIndex = topologicalIndex
			topologicalIndex++
			topologicalEvents = append(topologicalEvents, event)

			items = append(items, event)
			err := store.dbSetEvents([]*Event{event})
			if err != nil {
				t.Fatal(err)
			}
		}
		events[p.hex] = items
	}

	//check events where correctly inserted and can be retrieved
	for p, evs := range events {
		for k, ev := range evs {
			rev, err := store.dbGetEvent(ev.Hex())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ev.Body, rev.Body) {
				t.Fatalf("events[%s][%d].Body should be %#v, not %#v", p, k, ev.Body, rev.Body)
			}
			if !reflect.DeepEqual(ev.Signature, rev.Signature) {
				t.Fatalf("events[%s][%d].Signature should be %#v, not %#v", p, k, ev.Signature, rev.Signature)
			}
			if ver, err := rev.Verify(); err != nil && !ver {
				t.Fatalf("failed to verify signature. err: %s", err)
			}
		}
	}

	// check topological order of events was correctly created
	dbTopologicalEvents := []*Event{}
	index := 0
	batchSize := 100
	for {
		topologicalEvents, err := store.dbTopologicalEvents(index*batchSize, batchSize)
		if err != nil {
			t.Fatal(err)
		}
		dbTopologicalEvents = append(dbTopologicalEvents, topologicalEvents...)
		// Exit after the last batch
		if len(topologicalEvents) < batchSize {
			break
		}
		index++
	}

	if len(dbTopologicalEvents) != len(topologicalEvents) {
		t.Fatalf("Length of dbTopologicalEvents should be %d, not %d",
			len(topologicalEvents), len(dbTopologicalEvents))
	}
	for i, dte := range dbTopologicalEvents {
		te := topologicalEvents[i]

		if dte.Hex() != te.Hex() {
			t.Fatalf("dbTopologicalEvents[%d].Hex should be %s, not %s", i,
				te.Hex(),
				dte.Hex())
		}
		if !reflect.DeepEqual(te.Body, dte.Body) {
			t.Fatalf("dbTopologicalEvents[%d].Body should be %#v, not %#v", i,
				te.Body,
				dte.Body)
		}
		if !reflect.DeepEqual(te.Signature, dte.Signature) {
			t.Fatalf("dbTopologicalEvents[%d].Signature should be %#v, not %#v", i,
				te.Signature,
				dte.Signature)
		}

		if ver, err := dte.Verify(); err != nil && !ver {
			t.Fatalf("failed to verify signature. err: %s", err)
		}
	}
}

func TestDBRoundMethods(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	_, participants := initPeers(3)

	round := NewRoundInfo()
	events := make(map[string]*Event)
	for _, p := range participants {
		event := NewEvent([][]byte{},
			[]InternalTransaction{},
			[]BlockSignature{},
			[]string{"", ""},
			p.pubKey,
			0)
		events[p.hex] = event
		round.AddCreatedEvent(event.Hex(), true)
	}

	if err := store.dbSetRound(0, round); err != nil {
		t.Fatal(err)
	}

	storedRound, err := store.dbGetRound(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(round, storedRound) {
		t.Fatalf("Round and StoredRound do not match")
	}
}

func TestDBBlockMethods(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, participants := initPeers(3)

	index := 0
	roundReceived := 5
	transactions := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
		[]byte("tx4"),
		[]byte("tx5"),
	}
	internalTransactions := []InternalTransaction{
		NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1Pub", "paris", "peer1")),
		NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2Pub", "london", "peer2")),
	}
	frameHash := []byte("this is the frame hash")

	block := NewBlock(index, roundReceived, frameHash, peerSet.Peers, transactions, internalTransactions)

	receipts := []InternalTransactionReceipt{}
	for _, itx := range block.InternalTransactions() {
		receipts = append(receipts, itx.AsAccepted())
	}
	block.Body.InternalTransactionReceipts = receipts

	sig1, err := block.Sign(participants[0].privKey)
	if err != nil {
		t.Fatal(err)
	}

	sig2, err := block.Sign(participants[1].privKey)
	if err != nil {
		t.Fatal(err)
	}

	block.SetSignature(sig1)
	block.SetSignature(sig2)

	t.Run("Store Block", func(t *testing.T) {
		if err := store.dbSetBlock(block); err != nil {
			t.Fatal(err)
		}

		storedBlock, err := store.dbGetBlock(index)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(storedBlock.Body, block.Body) {
			t.Fatalf("Block and StoredBlock bodies do not match")
		}

		if !reflect.DeepEqual(storedBlock.Signatures, block.Signatures) {
			t.Fatalf("Block and StoredBlock signatures do not match")
		}
	})

	t.Run("Check signatures in stored Block", func(t *testing.T) {
		storedBlock, err := store.dbGetBlock(index)
		if err != nil {
			t.Fatal(err)
		}

		val1Sig, ok := storedBlock.Signatures[participants[0].hex]
		if !ok {
			t.Fatalf("Validator1 signature not stored in block")
		}
		if val1Sig != sig1.Signature {
			t.Fatal("Validator1 block signatures differ")
		}

		val2Sig, ok := storedBlock.Signatures[participants[1].hex]
		if !ok {
			t.Fatalf("Validator2 signature not stored in block")
		}
		if val2Sig != sig2.Signature {
			t.Fatal("Validator2 block signatures differ")
		}
	})
}

func TestDBFrameMethods(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, participants := initPeers(3)

	events := []*FrameEvent{}
	roots := make(map[string]*Root)
	for _, p := range participants {
		event := NewEvent(
			[][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], 0))},
			[]InternalTransaction{},
			[]BlockSignature{{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
			[]string{"", ""},
			p.pubKey,
			0)
		event.Sign(p.privKey)
		frameEvent := &FrameEvent{
			Core:             event,
			Round:            1,
			LamportTimestamp: 1,
			Witness:          true,
		}
		events = append(events, frameEvent)

		roots[p.hex] = NewRoot()
	}

	frame := &Frame{
		Round:  1,
		Peers:  peerSet.Peers,
		Events: events,
		Roots:  roots,
	}

	t.Run("Store Frame", func(t *testing.T) {
		if err := store.dbSetFrame(frame); err != nil {
			t.Fatal(err)
		}

		storedFrame, err := store.dbGetFrame(frame.Round)
		if err != nil {
			t.Fatal(err)
		}

		//force computing of IDs for DeepEqual
		for _, p := range storedFrame.Peers {
			p.ID()
		}

		if !reflect.DeepEqual(storedFrame, frame) {
			t.Fatalf("Frame and StoredFrame do not match")
		}
	})
}

/*******************************************************************************
Test wrapper methods work. These methods use the inmemStore as a cache on top of
the DB.
*******************************************************************************/

func TestBadgerPeerSets(t *testing.T) {
	cacheSize := 1000

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, _ := initPeers(3)

	err := store.SetPeerSet(0, peerSet)
	if err != nil {
		t.Fatal(err)
	}

	//check that it made it into the InmemStore
	iPeerSet, err := store.GetPeerSet(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(iPeerSet, peerSet) {
		t.Fatalf("InmemStore PeerSet should be %#v, not %#v", peerSet, iPeerSet)
	}

	//check that it also made it into the DB
	dPeerSet, err := store.GetPeerSet(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(dPeerSet, peerSet) {
		t.Fatalf("DB PeerSet should be %#v, not %#v", peerSet, dPeerSet)
	}

	//Check Repertoire and Roots
	repertoire, err := store.dbGetRepertoire()
	if err != nil {
		t.Fatal(err)
	}

	for pub, peer := range peerSet.ByPubKey {
		p, ok := repertoire[pub]
		if !ok {
			t.Fatalf("Repertoire[%s] not found", pub)
		}

		//force ID
		p.ID()

		if !reflect.DeepEqual(p, peer) {
			t.Fatalf("repertoire[%s] should be %#v, not %#v", pub, peer, p)
		}
	}
}

func TestBadgerEvents(t *testing.T) {
	//Insert more events than can fit in cache to test retrieving from db.
	cacheSize := 10
	testSize := 100

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, participants := initPeers(3)

	if err := store.SetPeerSet(0, peerSet); err != nil {
		t.Fatal(err)
	}

	//insert event
	events := make(map[string][]*Event)
	for _, p := range participants {
		items := []*Event{}
		for k := 0; k < testSize; k++ {
			event := NewEvent([][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], k))},
				[]InternalTransaction{},
				[]BlockSignature{{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
				[]string{"", ""},
				p.pubKey,
				k)
			items = append(items, event)
			err := store.SetEvent(event)
			if err != nil {
				t.Fatal(err)
			}
		}
		events[p.hex] = items
	}

	// check that events were correctly inserted
	for p, evs := range events {
		for k, ev := range evs {
			rev, err := store.GetEvent(ev.Hex())
			if err != nil {
				rev, err = store.dbGetEvent(ev.Hex())
			}
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(ev.Body, rev.Body) {
				t.Fatalf("events[%s][%d].Body should be %#v, not %#v", p, k, ev, rev)
			}
			if !reflect.DeepEqual(ev.Signature, rev.Signature) {
				t.Fatalf("events[%s][%d].Signature should be %#v, not %#v", p, k, ev.Signature, rev.Signature)
			}
		}
	}

	//check retrieving events per peerSet
	skipIndex := -1 //do not skip any indexes
	for _, p := range participants {
		pEvents, err := store.ParticipantEvents(p.hex, skipIndex)
		if err != nil {
			pEvents, err = store.dbParticipantEvents(p.hex, skipIndex)
		}
		if err != nil {
			t.Fatal(err)
		}
		if l := len(pEvents); l != testSize {
			t.Fatalf("%s should have %d events, not %d", p.hex, testSize, l)
		}

		expectedEvents := events[p.hex][skipIndex+1:]
		for k, e := range expectedEvents {
			if e.Hex() != pEvents[k] {
				t.Fatalf("peerSetEvents[%s][%d] should be %s, not %s",
					p.hex, k, e.Hex(), pEvents[k])
			}
		}
	}

	//check retrieving peerSet last
	for _, p := range participants {
		last, err := store.LastEventFrom(p.hex)
		if err != nil {
			t.Fatal(err)
		}

		evs := events[p.hex]
		expectedLast := evs[len(evs)-1]
		if last != expectedLast.Hex() {
			t.Fatalf("%s last should be %s, not %s", p.hex, expectedLast.Hex(), last)
		}
	}

	expectedKnown := make(map[uint32]int)
	for _, p := range participants {
		expectedKnown[p.id] = testSize - 1
	}
	known := store.KnownEvents()
	if !reflect.DeepEqual(expectedKnown, known) {
		t.Fatalf("Incorrect Known. Got %#v, expected %#v", known, expectedKnown)
	}

	for _, p := range participants {
		evs := events[p.hex]
		for _, ev := range evs {
			if err := store.AddConsensusEvent(ev); err != nil {
				t.Fatal(err)
			}
		}

	}
}

func TestBadgerRounds(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, participants := initPeers(3)

	if err := store.SetPeerSet(0, peerSet); err != nil {
		t.Fatal(err)
	}

	round := NewRoundInfo()
	events := make(map[string]*Event)
	for _, p := range participants {
		event := NewEvent([][]byte{},
			[]InternalTransaction{},
			[]BlockSignature{},
			[]string{"", ""},
			p.pubKey,
			0)
		events[p.hex] = event
		round.AddCreatedEvent(event.Hex(), true)
	}

	if err := store.SetRound(0, round); err != nil {
		t.Fatal(err)
	}

	if c := store.LastRound(); c != 0 {
		t.Fatalf("Store LastRound should be 0, not %d", c)
	}

	storedRound, err := store.GetRound(0)
	if err != nil {
		storedRound, err = store.dbGetRound(0)
	}
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(round, storedRound) {
		t.Fatalf("Round and StoredRound do not match")
	}
}

func TestBadgerBlocks(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, participants := initPeers(3)

	if err := store.SetPeerSet(0, peerSet); err != nil {
		t.Fatal(err)
	}

	index := 0
	roundReceived := 5
	transactions := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
		[]byte("tx4"),
		[]byte("tx5"),
	}
	internalTransactions := []InternalTransaction{
		NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1", "paris", "peer1")),
		NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2", "london", "peer2")),
	}
	frameHash := []byte("this is the frame hash")
	block := NewBlock(index, roundReceived, frameHash, []*peers.Peer{}, transactions, internalTransactions)

	receipts := []InternalTransactionReceipt{}
	for _, itx := range block.InternalTransactions() {
		receipts = append(receipts, itx.AsAccepted())
	}
	block.Body.InternalTransactionReceipts = receipts

	sig1, err := block.Sign(participants[0].privKey)
	if err != nil {
		t.Fatal(err)
	}

	sig2, err := block.Sign(participants[1].privKey)
	if err != nil {
		t.Fatal(err)
	}

	block.SetSignature(sig1)
	block.SetSignature(sig2)

	t.Run("Store Block", func(t *testing.T) {
		if err := store.SetBlock(block); err != nil {
			t.Fatal(err)
		}

		storedBlock, err := store.GetBlock(index)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(storedBlock.Body, block.Body) {
			t.Fatalf("Block and StoredBlock bodies do not match")
		}

		if !reflect.DeepEqual(storedBlock.Signatures, block.Signatures) {
			t.Fatalf("Block and StoredBlock signatures do not match")
		}
	})

	t.Run("Check signatures in stored Block", func(t *testing.T) {
		storedBlock, err := store.GetBlock(index)
		if err != nil {
			t.Fatal(err)
		}

		val1Sig, ok := storedBlock.Signatures[participants[0].hex]
		if !ok {
			t.Fatalf("Validator1 signature not stored in block")
		}
		if val1Sig != sig1.Signature {
			t.Fatal("Validator1 block signatures differ")
		}

		val2Sig, ok := storedBlock.Signatures[participants[1].hex]
		if !ok {
			t.Fatalf("Validator2 signature not stored in block")
		}
		if val2Sig != sig2.Signature {
			t.Fatal("Validator2 block signatures differ")
		}
	})
}

func TestBadgerFrames(t *testing.T) {
	cacheSize := 0

	store := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	peerSet, participants := initPeers(3)

	if err := store.SetPeerSet(0, peerSet); err != nil {
		t.Fatal(err)
	}

	events := []*FrameEvent{}
	roots := make(map[string]*Root)
	for _, p := range participants {
		event := NewEvent(
			[][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], 0))},
			[]InternalTransaction{},
			[]BlockSignature{{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
			[]string{"", ""},
			p.pubKey,
			0)
		event.Sign(p.privKey)
		frameEvent := &FrameEvent{
			Core: event,
		}
		events = append(events, frameEvent)

		roots[p.hex] = NewRoot()
	}

	frame := &Frame{
		Round:  1,
		Events: events,
		Roots:  roots,
	}

	t.Run("Store Frame", func(t *testing.T) {
		if err := store.SetFrame(frame); err != nil {
			t.Fatal(err)
		}

		storedFrame, err := store.GetFrame(frame.Round)
		if err != nil {
			storedFrame, err = store.dbGetFrame(frame.Round)
		}
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(storedFrame, frame) {
			t.Fatalf("Frame and StoredFrame do not match")
		}
	})
}
