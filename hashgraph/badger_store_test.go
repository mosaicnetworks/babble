package hashgraph

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/crypto"
)

func initBadgerStore(cacheSize int, t *testing.T) (*BadgerStore, []pub) {
	n := 3
	participantPubs := []pub{}
	participants := make(map[string]int)
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pubKey := crypto.FromECDSAPub(&key.PublicKey)
		participantPubs = append(participantPubs,
			pub{i, key, pubKey, fmt.Sprintf("0x%X", pubKey)})
		participants[fmt.Sprintf("0x%X", pubKey)] = i
	}

	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)
	dir, err := ioutil.TempDir("test_data", "badger")
	if err != nil {
		log.Fatal(err)
	}

	store, err := NewBadgerStore(participants, cacheSize, dir)
	if err != nil {
		t.Fatal(err)
	}

	return store, participantPubs
}

func removeBadgerStore(store *BadgerStore, t *testing.T) {
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(store.path); err != nil {
		t.Fatal(err)
	}
}

func createTestDB(dir string, t *testing.T) *BadgerStore {
	participants := map[string]int{
		"alice":   0,
		"bob":     1,
		"charlie": 2,
	}
	cacheSize := 100

	store, err := NewBadgerStore(participants, cacheSize, dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return store
}

func TestNewBadgerStore(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	dbPath := "test_data/badger"
	store := createTestDB(dbPath, t)
	defer os.RemoveAll(store.path)

	if store.path != dbPath {
		t.Fatalf("unexpected path %q", store.path)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("err: %s", err)
	}

	//check roots
	inmemRoots := store.inmemStore.rootsByParticipant
	for participant, root := range inmemRoots {
		dbRoot, err := store.dbGetRoot(participant)
		if err != nil {
			t.Fatalf("Error retrieving DB root for participant %s: %s", participant, err)
		}
		if !reflect.DeepEqual(dbRoot, root) {
			t.Fatalf("%s DB root should be %#v, not %#v", participant, root, dbRoot)
		}
	}

	if err := store.Close(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestLoadBadgerStore(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)
	dbPath := "test_data/badger"

	//Create the test db
	tempStore := createTestDB(dbPath, t)
	defer os.RemoveAll(tempStore.path)
	tempStore.Close()

	badgerStore, err := LoadBadgerStore(cacheSize, tempStore.path)
	if err != nil {
		t.Fatal(err)
	}

	dbParticipants, err := badgerStore.dbGetParticipants()
	if err != nil {
		t.Fatal(err)
	}

	if len(badgerStore.participants) != len(dbParticipants) {
		t.Fatalf("store.participants should contain %d items, not %d",
			len(dbParticipants),
			len(badgerStore.participants))
	}

	for dbP, dbID := range dbParticipants {
		id, ok := badgerStore.participants[dbP]
		if !ok {
			t.Fatalf("BadgerStore participants does not contains %s", dbP)
		}
		if id != dbID {
			t.Fatalf("participant %s ID should be %d, not %d", dbP, dbID, id)
		}
	}

}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Call DB methods directly

func TestDBEventMethods(t *testing.T) {
	cacheSize := 0
	testSize := 100
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	//insert events in db directly
	events := make(map[string][]Event)
	topologicalIndex := 0
	topologicalEvents := []Event{}
	for _, p := range participants {
		items := []Event{}
		for k := 0; k < testSize; k++ {
			event := NewEvent(
				[][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], k))},
				[]BlockSignature{BlockSignature{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
				[]string{"", ""},
				p.pubKey,
				k)
			event.Sign(p.privKey)
			event.topologicalIndex = topologicalIndex
			topologicalIndex++
			topologicalEvents = append(topologicalEvents, event)

			items = append(items, event)
			err := store.dbSetEvents([]Event{event})
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

	//check topological order of events was correctly created
	dbTopologicalEvents, err := store.dbTopologicalEvents()
	if err != nil {
		t.Fatal(err)
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

	//check that participant events where correctly added
	skipIndex := -1 //do not skip any indexes
	for _, p := range participants {
		pEvents, err := store.dbParticipantEvents(p.hex, skipIndex)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(pEvents); l != testSize {
			t.Fatalf("%s should have %d events, not %d", p.hex, testSize, l)
		}

		expectedEvents := events[p.hex][skipIndex+1:]
		for k, e := range expectedEvents {
			if e.Hex() != pEvents[k] {
				t.Fatalf("ParticipantEvents[%s][%d] should be %s, not %s",
					p.hex, k, e.Hex(), pEvents[k])
			}
		}
	}
}

func TestDBRoundMethods(t *testing.T) {
	cacheSize := 0
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	round := NewRoundInfo()
	events := make(map[string]Event)
	for _, p := range participants {
		event := NewEvent([][]byte{},
			[]BlockSignature{},
			[]string{"", ""},
			p.pubKey,
			0)
		events[p.hex] = event
		round.AddEvent(event.Hex(), true)
	}

	if err := store.dbSetRound(0, *round); err != nil {
		t.Fatal(err)
	}

	storedRound, err := store.dbGetRound(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*round, storedRound) {
		t.Fatalf("Round and StoredRound do not match")
	}

	witnesses := store.RoundWitnesses(0)
	expectedWitnesses := round.Witnesses()
	if len(witnesses) != len(expectedWitnesses) {
		t.Fatalf("There should be %d witnesses, not %d", len(expectedWitnesses), len(witnesses))
	}
	for _, w := range expectedWitnesses {
		if !contains(witnesses, w) {
			t.Fatalf("Witnesses should contain %s", w)
		}
	}
}

func TestDBParticipantMethods(t *testing.T) {
	cacheSize := 0
	store, _ := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	if err := store.dbSetParticipants(store.participants); err != nil {
		t.Fatal(err)
	}

	participantsFromDB, err := store.dbGetParticipants()
	if err != nil {
		t.Fatal(err)
	}

	for p, id := range store.participants {
		dbID, ok := participantsFromDB[p]
		if !ok {
			t.Fatalf("DB does not contain participant %s", p)
		}
		if dbID != id {
			t.Fatalf("DB participant %s should have ID %d, not %d", p, id, dbID)
		}
	}
}

func TestDBBlockMethods(t *testing.T) {
	cacheSize := 0
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	index := 0
	roundReceived := 5
	transactions := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
		[]byte("tx4"),
		[]byte("tx5"),
	}
	frameHash := []byte("this is the frame hash")

	block := NewBlock(index, roundReceived, frameHash, transactions)

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

		if !reflect.DeepEqual(storedBlock, block) {
			t.Fatalf("Block and StoredBlock do not match")
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
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	events := []Event{}
	roots := []Root{}
	for id, p := range participants {
		event := NewEvent(
			[][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], 0))},
			[]BlockSignature{BlockSignature{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
			[]string{"", ""},
			p.pubKey,
			0)
		event.Sign(p.privKey)
		events = append(events, event)

		root := NewBaseRoot(id)
		roots = append(roots, root)
	}
	frame := Frame{
		Round:  1,
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

		if !reflect.DeepEqual(storedFrame, frame) {
			t.Fatalf("Frame and StoredFrame do not match")
		}
	})
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Check that the wrapper methods work
//These methods use the inmemStore as a cache on top of the DB

func TestBadgerEvents(t *testing.T) {
	//Insert more events than can fit in cache to test retrieving from db.
	cacheSize := 10
	testSize := 100
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	//insert event
	events := make(map[string][]Event)
	for _, p := range participants {
		items := []Event{}
		for k := 0; k < testSize; k++ {
			event := NewEvent([][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], k))},
				[]BlockSignature{BlockSignature{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
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

	// check that events were correclty inserted
	for p, evs := range events {
		for k, ev := range evs {
			rev, err := store.GetEvent(ev.Hex())
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

	//check retrieving events per participant
	skipIndex := -1 //do not skip any indexes
	for _, p := range participants {
		pEvents, err := store.ParticipantEvents(p.hex, skipIndex)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(pEvents); l != testSize {
			t.Fatalf("%s should have %d events, not %d", p.hex, testSize, l)
		}

		expectedEvents := events[p.hex][skipIndex+1:]
		for k, e := range expectedEvents {
			if e.Hex() != pEvents[k] {
				t.Fatalf("ParticipantEvents[%s][%d] should be %s, not %s",
					p.hex, k, e.Hex(), pEvents[k])
			}
		}
	}

	//check retrieving participant last
	for _, p := range participants {
		last, _, err := store.LastEventFrom(p.hex)
		if err != nil {
			t.Fatal(err)
		}

		evs := events[p.hex]
		expectedLast := evs[len(evs)-1]
		if last != expectedLast.Hex() {
			t.Fatalf("%s last should be %s, not %s", p.hex, expectedLast.Hex(), last)
		}
	}

	expectedKnown := make(map[int]int)
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
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	round := NewRoundInfo()
	events := make(map[string]Event)
	for _, p := range participants {
		event := NewEvent([][]byte{},
			[]BlockSignature{},
			[]string{"", ""},
			p.pubKey,
			0)
		events[p.hex] = event
		round.AddEvent(event.Hex(), true)
	}

	if err := store.SetRound(0, *round); err != nil {
		t.Fatal(err)
	}

	if c := store.LastRound(); c != 0 {
		t.Fatalf("Store LastRound should be 0, not %d", c)
	}

	storedRound, err := store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*round, storedRound) {
		t.Fatalf("Round and StoredRound do not match")
	}

	witnesses := store.RoundWitnesses(0)
	expectedWitnesses := round.Witnesses()
	if len(witnesses) != len(expectedWitnesses) {
		t.Fatalf("There should be %d witnesses, not %d", len(expectedWitnesses), len(witnesses))
	}
	for _, w := range expectedWitnesses {
		if !contains(witnesses, w) {
			t.Fatalf("Witnesses should contain %s", w)
		}
	}
}

func TestBadgerBlocks(t *testing.T) {
	cacheSize := 0
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	index := 0
	roundReceived := 5
	transactions := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
		[]byte("tx4"),
		[]byte("tx5"),
	}
	frameHash := []byte("this is the frame hash")
	block := NewBlock(index, roundReceived, frameHash, transactions)

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

		if !reflect.DeepEqual(storedBlock, block) {
			t.Fatalf("Block and StoredBlock do not match")
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
	store, participants := initBadgerStore(cacheSize, t)
	defer removeBadgerStore(store, t)

	events := []Event{}
	roots := []Root{}
	for id, p := range participants {
		event := NewEvent(
			[][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], 0))},
			[]BlockSignature{BlockSignature{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
			[]string{"", ""},
			p.pubKey,
			0)
		event.Sign(p.privKey)
		events = append(events, event)

		root := NewBaseRoot(id)
		roots = append(roots, root)
	}
	frame := Frame{
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
			t.Fatal(err)
		}

		if !reflect.DeepEqual(storedFrame, frame) {
			t.Fatalf("Frame and StoredFrame do not match")
		}
	})
}
