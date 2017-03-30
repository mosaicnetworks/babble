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
)

func TestEventCache(t *testing.T) {
	size := 10
	cache := NewEventCache(size)
	events := []Event{}

	for i := 0; i < size; i++ {
		e := NewEvent([][]byte{}, []string{"", ""}, []byte(fmt.Sprintf("a%d", i)))
		events = append(events, e)
		cache.Set(e)
	}
	if ct := cache.count; ct != size {
		t.Fatalf("EventCache count should be %d, not %d", size, ct)
	}
	if tt := cache.tot; tt != size {
		t.Fatalf("EventCache tot should be %d, not %d", size, tt)
	}
	if l := len(cache.buffer); l != 0 {
		t.Fatalf("EventCache buffer should contain 0 items, not %d", l)
	}
	if l := len(cache.index); l != size {
		t.Fatalf("EventCache index should contain %d items, not %d", size, l)
	}
	for i := 0; i < size; i++ {
		expected := events[i]
		cached, ok := cache.Get(expected.Hex())
		if !ok {
			t.Fatalf("Unable to retrieve event %d", i)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}

	for i := 0; i < size; i++ {
		e := NewEvent([][]byte{}, []string{"", ""}, []byte(fmt.Sprintf("b%d", i)))
		events = append(events, e)
		cache.Set(e)
	}
	if ct := cache.count; ct != size {
		t.Fatalf("EventCache count should be %d, not %d", size, ct)
	}
	if tt := cache.tot; tt != 2*size {
		t.Fatalf("EventCache tot should be %d, not %d", 2*size, tt)
	}
	if l := len(cache.buffer); l != size {
		t.Fatalf("EventCache buffer should contain %d items, not %d", size, l)
	}
	if l := len(cache.index); l != size {
		t.Fatalf("EventCache index should contain %d items, not %d", size, l)
	}

	for i := 0; i < 2*size; i++ {
		expected := events[i]
		cached, ok := cache.Get(expected.Hex())
		if !ok {
			t.Fatalf("Unable to retrieve event %d", i)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}

	for i := 0; i < size; i++ {
		e := NewEvent([][]byte{}, []string{"", ""}, []byte(fmt.Sprintf("c%d", i)))
		events = append(events, e)
		cache.Set(e)
	}
	if ct := cache.count; ct != size {
		t.Fatalf("EventCache count should be %d, not %d", size, ct)
	}
	if tt := cache.tot; tt != 3*size {
		t.Fatalf("EventCache tot should be %d, not %d", 2*size, tt)
	}
	if l := len(cache.buffer); l != size {
		t.Fatalf("EventCache buffer should contain %d items, not %d", size, l)
	}
	if l := len(cache.index); l != size {
		t.Fatalf("EventCache index should contain %d items, not %d", size, l)
	}
	for i := 0; i < size; i++ {
		notExpected := events[i]
		_, ok := cache.Get(notExpected.Hex())
		if ok {
			t.Fatalf("event %d should have been deleted", i)
		}
	}
	for i := size; i < 3*size; i++ {
		expected := events[i]
		cached, ok := cache.Get(expected.Hex())
		if !ok {
			t.Fatalf("Unable to retrieve event %d", i)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}
}

func TestRoundCache(t *testing.T) {
	size := 10
	cache := NewRoundCache(size)
	rounds := []*RoundInfo{}

	for i := 0; i < size; i++ {
		r := NewRoundInfo()
		rounds = append(rounds, r)
		cache.Set(i, *r)
	}
	if ct := cache.count; ct != size {
		t.Fatalf("RoundCache count should be %d, not %d", size, ct)
	}
	if tt := cache.tot; tt != size {
		t.Fatalf("RoundCache tot should be %d, not %d", size, tt)
	}
	if l := len(cache.buffer); l != 0 {
		t.Fatalf("RoundCache buffer should contain 0 items, not %d", l)
	}
	if l := len(cache.index); l != size {
		t.Fatalf("RoundCache index should contain %d items, not %d", size, l)
	}
	for i := 0; i < size; i++ {
		expected := rounds[i]
		cached, ok := cache.Get(i)
		if !ok {
			t.Fatalf("Unable to retrieve round %d", i)
		}
		if !reflect.DeepEqual(*expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}

	for i := 0; i < size; i++ {
		r := NewRoundInfo()
		rounds = append(rounds, r)
		cache.Set(size+i, *r)
	}
	if ct := cache.count; ct != size {
		t.Fatalf("RoundCache count should be %d, not %d", size, ct)
	}
	if tt := cache.tot; tt != 2*size {
		t.Fatalf("RoundCache tot should be %d, not %d", 2*size, tt)
	}
	if l := len(cache.buffer); l != size {
		t.Fatalf("RoundCache buffer should contain %d items, not %d", size, l)
	}
	if l := len(cache.index); l != size {
		t.Fatalf("RoundCache index should contain %d items, not %d", size, l)
	}

	for i := 0; i < 2*size; i++ {
		expected := rounds[i]
		cached, ok := cache.Get(i)
		if !ok {
			t.Fatalf("Unable to retrieve round %d", i)
		}
		if !reflect.DeepEqual(*expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}

	for i := 0; i < size; i++ {
		r := NewRoundInfo()
		rounds = append(rounds, r)
		cache.Set(2*size+i, *r)
	}
	if ct := cache.count; ct != size {
		t.Fatalf("RoundCache count should be %d, not %d", size, ct)
	}
	if tt := cache.tot; tt != 3*size {
		t.Fatalf("RoundCache tot should be %d, not %d", 2*size, tt)
	}
	if l := len(cache.buffer); l != size {
		t.Fatalf("RoundCache buffer should contain %d items, not %d", size, l)
	}
	if l := len(cache.index); l != size {
		t.Fatalf("RoundCache index should contain %d items, not %d", size, l)
	}
	for i := 0; i < size; i++ {
		_, ok := cache.Get(i)
		if ok {
			t.Fatalf("round %d should have been deleted", i)
		}
	}
	for i := size; i < 3*size; i++ {
		expected := rounds[i]
		cached, ok := cache.Get(i)
		if !ok {
			t.Fatalf("Unable to retrieve round %d", i)
		}
		if !reflect.DeepEqual(*expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}
}

func TestStringListCache(t *testing.T) {
	size := 10
	cache := NewStringListCache(size)
	items := []string{}

	for i := 0; i < size; i++ {
		e := fmt.Sprintf("a%d", i)
		cache.Set(e)
		items = append(items, e)
	}
	if tt := cache.tot; tt != size {
		t.Fatalf("StringListCache tot should be %d, not %d", size, tt)
	}
	if l := len(cache.items); l != size {
		t.Fatalf("StringListCache should contain %d items, not %d", size, l)
	}

	cached, tot := cache.Get()
	if l := len(cached); l != size {
		t.Fatalf("StringListCache Get() should return %d items, not %d", size, l)
	}
	if tot != size {
		t.Fatalf("StringListCache Get() should return %d tot, not %d", size, tot)
	}

	for i := 0; i < size; i++ {
		expected := items[i]
		cached := cached[i]
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", i)
		}
	}

	for i := 0; i < size; i++ {
		e := fmt.Sprintf("b%d", i)
		cache.Set(e)
		items = append(items, e)
	}
	if tt := cache.tot; tt != 2*size {
		t.Fatalf("StringListCache tot should be %d, not %d", 2*size, tt)
	}
	if l := len(cache.items); l != size {
		t.Fatalf("StringListCache should contain %d items, not %d", size, l)
	}

	cached, tot = cache.Get()
	if l := len(cached); l != size {
		t.Fatalf("StringListCache Get() should return %d items, not %d", size, l)
	}
	if tot != 2*size {
		t.Fatalf("StringListCache Get() should return %d tot, not %d", 2*size, tot)
	}

	for i := 0; i < size; i++ {
		expected := items[size+i]
		cached := cached[i]
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("index %d: expected and cached are not equal", size+i)
		}
	}
}

func TestParticipantEventsCache(t *testing.T) {
	size := 10
	participants := []string{"alice", "bob", "charlie"}
	pec := NewParticipantEventsCache(size, participants)
	participantEvents := make(map[string][]string)

	for i, p := range participants {
		es := []string{}
		for j := 0; j < size; j++ {
			e := fmt.Sprintf("p%d%d", i, j)
			es = append(es, e)
			pec.Set(p, e)
		}
		participantEvents[p] = es
	}

	known := pec.Known()
	expectedKnown := make(map[string]int)
	for _, p := range participants {
		expectedKnown[p] = size
	}
	if !reflect.DeepEqual(known, expectedKnown) {
		t.Fatalf("Known() expected: %#v, actual: %#v", expectedKnown, known)
	}

	for _, p := range participants {
		items := participantEvents[p]
		expectedLast := items[len(items)-1]
		last, err := pec.GetLast(p)
		if err != nil {
			t.Fatal(err)
		}
		if last != expectedLast {
			t.Fatalf("%s's last event should be %s, not %s", p, expectedLast, last)
		}
	}

	for _, p := range participants {
		skipped5, err := pec.Get(p, 5)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(skipped5); l != 5 {
			t.Fatalf("skipped5 should contain 5 items, not %d", l)
		}
		items := participantEvents[p]
		for i := 0; i < 5; i++ {
			if items[5+i] != skipped5[i] {
				t.Fatalf("skipped5[%d] should be %s, not %s", i, items[5+i], skipped5[i])
			}
		}

	}

	//Test Rolling
	pec = NewParticipantEventsCache(size, participants)
	participantEvents = make(map[string][]string)
	for i, p := range participants {
		es := []string{}
		for j := 0; j < 2*size; j++ {
			e := fmt.Sprintf("p%d%d", i, j)
			es = append(es, e)
			pec.Set(p, e)
		}
		participantEvents[p] = es
	}

	known = pec.Known()
	expectedKnown = make(map[string]int)
	for _, p := range participants {
		expectedKnown[p] = 2 * size
	}
	if !reflect.DeepEqual(known, expectedKnown) {
		t.Fatalf("Known() expected: %#v, actual: %#v", expectedKnown, known)
	}

	for _, p := range participants {
		_, err := pec.Get(p, 5)
		if err == nil || err != ErrTooLate {
			t.Fatalf("skipping 5 should return TooLateErr")
		}

		skipped12, err := pec.Get(p, 12)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(skipped12); l != 8 {
			t.Fatalf("skipped12 should contain 8 items, not %d", l)
		}
		items := participantEvents[p]
		for i := 0; i < 8; i++ {
			if items[12+i] != skipped12[i] {
				t.Fatalf("skipped12[%d] should be %s, not %s", i, items[12+i], skipped12[i])
			}
		}
	}

}
