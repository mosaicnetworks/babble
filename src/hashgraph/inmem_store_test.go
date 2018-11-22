package hashgraph

import (
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

type participant struct {
	id      int
	privKey *ecdsa.PrivateKey
	pubKey  []byte
	hex     string
}

func initInmemStore(cacheSize int) (*InmemStore, []participant) {
	n := 3
	participants := []participant{}

	pirs := []*peers.Peer{}
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pubKey := crypto.FromECDSAPub(&key.PublicKey)
		peer := peers.NewPeer(fmt.Sprintf("0x%X", pubKey), "")
		participants = append(participants,
			participant{peer.ID, key, pubKey, peer.PubKeyHex})
		pirs = append(pirs, peer)
	}

	peerSet := peers.NewPeerSet(pirs)

	store := NewInmemStore(peerSet, cacheSize)

	return store, participants
}

func TestInmemEvents(t *testing.T) {
	cacheSize := 100
	testSize := 15
	store, participants := initInmemStore(cacheSize)

	events := make(map[string][]*Event)

	t.Run("Store Events", func(t *testing.T) {
		for _, p := range participants {
			items := []*Event{}
			for k := 0; k < testSize; k++ {
				event := NewEvent([][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], k))},
					nil,
					[]BlockSignature{BlockSignature{Validator: []byte("validator"), Index: 0, Signature: "r|s"}},
					[]string{"", ""},
					p.pubKey,
					k)
				_ = event.Hex() //just to set private variables
				items = append(items, event)
				err := store.SetEvent(event)
				if err != nil {
					t.Fatal(err)
				}
			}
			events[p.hex] = items
		}

		for p, evs := range events {
			for k, ev := range evs {
				rev, err := store.GetEvent(ev.Hex())
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(ev.Body, rev.Body) {
					t.Fatalf("events[%s][%d] should be %#v, not %#v", p, k, ev, rev)
				}
			}
		}
	})

	t.Run("Check ParticipantEventsCache", func(t *testing.T) {
		skipIndex := -1 //do not skip any indexes
		for _, p := range participants {
			pEvents, err := store.ParticipantEvents(p.hex, skipIndex)
			if err != nil {
				t.Fatal(err)
			}
			if l := len(pEvents); l != testSize {
				t.Fatalf("%s should have %d Events, not %d", p.hex, testSize, l)
			}

			expectedEvents := events[p.hex][skipIndex+1:]
			for k, e := range expectedEvents {
				if e.Hex() != pEvents[k] {
					t.Fatalf("ParticipantEvents[%s][%d] should be %s, not %s",
						p.hex, k, e.Hex(), pEvents[k])
				}
			}
		}
	})

	t.Run("Check KnownEvents", func(t *testing.T) {
		expectedKnown := make(map[int]int)
		for _, p := range participants {
			expectedKnown[p.id] = testSize - 1
		}
		known := store.KnownEvents()
		if !reflect.DeepEqual(expectedKnown, known) {
			t.Fatalf("Incorrect Known. Got %#v, expected %#v", known, expectedKnown)
		}
	})

	t.Run("Add ConsensusEvents", func(t *testing.T) {
		for _, p := range participants {
			evs := events[p.hex]
			for _, ev := range evs {
				if err := store.AddConsensusEvent(ev); err != nil {
					t.Fatal(err)
				}
			}
		}
	})

}

func TestInmemRounds(t *testing.T) {
	store, participants := initInmemStore(10)

	round := NewRoundInfo() //XXX
	events := make(map[string]*Event)
	for _, p := range participants {
		event := NewEvent([][]byte{},
			nil,
			[]BlockSignature{},
			[]string{"", ""},
			p.pubKey,
			0)
		events[p.hex] = event
		round.AddCreatedEvent(event.Hex(), true)
	}

	t.Run("Store Round", func(t *testing.T) {
		if err := store.SetRound(0, round); err != nil {
			t.Fatal(err)
		}
		storedRound, err := store.GetRound(0)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(round, storedRound) {
			t.Fatalf("Round and StoredRound do not match")
		}
	})

	t.Run("Check LastRound", func(t *testing.T) {
		if c := store.LastRound(); c != 0 {
			t.Fatalf("Store LastRound should be 0, not %d", c)
		}
	})

	t.Run("Check witnesses", func(t *testing.T) {
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
	})
}

func TestInmemBlocks(t *testing.T) {
	store, participants := initInmemStore(10)

	index := 0
	roundReceived := 7
	transactions := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
		[]byte("tx4"),
		[]byte("tx5"),
	}
	internalTransactions := []InternalTransaction{
		NewInternalTransaction(PEER_ADD, *peers.NewPeer("peer1", "paris")),
		NewInternalTransaction(PEER_REMOVE, *peers.NewPeer("peer2", "london")),
	}
	frameHash := []byte("this is the frame hash")

	block := NewBlock(index, roundReceived, frameHash, []*peers.Peer{}, transactions, internalTransactions)

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
