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

import "testing"
import "fmt"
import "reflect"

func TestParticipantEventsCache(t *testing.T) {
	size := 10
	testSize := 25
	participants := []string{"alice", "bob", "charlie"}
	pec := NewParticipantEventsCache(size, participants)

	items := make(map[string][]string)
	for _, p := range participants {
		items[p] = []string{}
	}

	for i := 0; i < testSize; i++ {
		for _, p := range participants {
			item := fmt.Sprintf("%s%d", p, i)

			pec.Add(p, item)

			pitems := items[p]
			pitems = append(pitems, item)
			items[p] = pitems
		}
	}

	known := pec.Known()
	for p, k := range known {
		if k != testSize {
			t.Errorf("Known[%s] should be %d, not %d", p, testSize, k)
		}
	}

	for _, p := range participants {
		if _, err := pec.Get(p, 0); err != nil && err != ErrTooLate {
			t.Fatalf("Skipping 0 elements should return ErrNotFatal")
		}

		skip := 10
		expected := items[p][skip:]
		cached, err := pec.Get(p, skip)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected and cached not equal")
		}

		skip2 := 15
		expected2 := items[p][skip2:]
		cached2, err := pec.Get(p, skip2)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected2, cached2) {
			t.Fatalf("expected and cached not equal")
		}

		skip3 := 27
		expected3 := []string{}
		cached3, err := pec.Get(p, skip3)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected3, cached3) {
			t.Fatalf("expected and cached not equal")
		}
	}
}

func TestParticipantEventsCacheEdge(t *testing.T) {
	size := 10
	testSize := 11
	participants := []string{"alice", "bob", "charlie"}
	pec := NewParticipantEventsCache(size, participants)

	items := make(map[string][]string)
	for _, p := range participants {
		items[p] = []string{}
	}

	for i := 0; i < testSize; i++ {
		for _, p := range participants {
			item := fmt.Sprintf("%s%d", p, i)

			pec.Add(p, item)

			pitems := items[p]
			pitems = append(pitems, item)
			items[p] = pitems
		}
	}

	for _, p := range participants {
		skip := size
		expected := items[p][skip:]
		cached, err := pec.Get(p, skip)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected and cached not equal")
		}
	}
}
