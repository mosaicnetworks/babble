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
package hashgraph

import (
	"fmt"
	"reflect"
	"testing"

	"bitbucket.org/mosaicnet/babble/crypto"
)

type pub struct {
	id     int
	pubKey []byte
	hex    string
}

func initInmemStore(cacheSize int) (*InmemStore, []pub) {
	n := 3
	participantPubs := []pub{}
	participants := make(map[string]int)
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pubKey := crypto.FromECDSAPub(&key.PublicKey)
		participantPubs = append(participantPubs,
			pub{i, pubKey, fmt.Sprintf("0x%X", pubKey)})
		participants[fmt.Sprintf("0x%X", pubKey)] = i
	}

	store := NewInmemStore(participants, cacheSize)
	return store, participantPubs
}

func TestInmemEvents(t *testing.T) {
	cacheSize := 100
	testSize := 15
	store, participants := initInmemStore(cacheSize)

	events := make(map[string][]Event)
	for _, p := range participants {
		items := []Event{}
		for k := 0; k < testSize; k++ {
			event := NewEvent([][]byte{[]byte(fmt.Sprintf("%s_%d", p.hex[:5], k))},
				[]string{"", ""}, p.pubKey, k)
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

	skip := 0
	for _, p := range participants {
		pEvents, err := store.ParticipantEvents(p.hex, skip)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(pEvents); l != testSize {
			t.Fatalf("%s should have %d events, not %d", p.hex, testSize, l)
		}

		expectedEvents := events[p.hex][skip:]
		for k, e := range expectedEvents {
			if e.Hex() != pEvents[k] {
				t.Fatalf("ParticipantEvents[%s][%d] should be %s, not %s",
					p.hex, k, e.Hex(), pEvents[k])
			}
		}
	}

	expectedKnown := make(map[int]int)
	for _, p := range participants {
		expectedKnown[p.id] = testSize
	}
	known := store.Known()
	if !reflect.DeepEqual(expectedKnown, known) {
		t.Fatalf("Incorrect Known. Got %#v, expected %#v", known, expectedKnown)
	}

	for _, p := range participants {
		evs := events[p.hex]
		for _, ev := range evs {
			if err := store.AddConsensusEvent(ev.Hex()); err != nil {
				t.Fatal(err)
			}
		}

	}
}

func TestInmemRounds(t *testing.T) {
	store, participants := initInmemStore(10)

	round := NewRoundInfo()
	events := make(map[string]Event)
	for _, p := range participants {
		event := NewEvent([][]byte{}, []string{"", ""}, p.pubKey, 0)
		events[p.hex] = event
		round.AddEvent(event.Hex(), true)
	}

	if err := store.SetRound(0, *round); err != nil {
		t.Fatal(err)
	}

	if c := store.Rounds(); c != 1 {
		t.Fatalf("Store should count 1 round, not %d", c)
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
