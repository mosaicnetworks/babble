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
	participants := map[string]int{
		"alice":   0,
		"bob":     1,
		"charlie": 2,
	}
	pec := NewParticipantEventsCache(size, participants)

	items := make(map[string][]string)
	for pk := range participants {
		items[pk] = []string{}
	}

	for i := 0; i < testSize; i++ {
		for pk := range participants {
			item := fmt.Sprintf("%s%d", pk, i)

			pec.Add(pk, item)

			pitems := items[pk]
			pitems = append(pitems, item)
			items[pk] = pitems
		}
	}

	known := pec.Known()
	for p, k := range known {
		if k != testSize {
			t.Errorf("Known[%s] should be %d, not %d", p, testSize, k)
		}
	}

	for pk := range participants {
		if _, err := pec.Get(pk, 0); err != nil && err != ErrTooLate {
			t.Fatalf("Skipping 0 elements should return ErrNotFatal")
		}

		skip := 10
		expected := items[pk][skip:]
		cached, err := pec.Get(pk, skip)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected and cached not equal")
		}

		skip2 := 15
		expected2 := items[pk][skip2:]
		cached2, err := pec.Get(pk, skip2)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected2, cached2) {
			t.Fatalf("expected and cached not equal")
		}

		skip3 := 27
		expected3 := []string{}
		cached3, err := pec.Get(pk, skip3)
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
	participants := map[string]int{
		"alice":   0,
		"bob":     1,
		"charlie": 2,
	}
	pec := NewParticipantEventsCache(size, participants)

	items := make(map[string][]string)
	for pk := range participants {
		items[pk] = []string{}
	}

	for i := 0; i < testSize; i++ {
		for pk := range participants {
			item := fmt.Sprintf("%s%d", pk, i)

			pec.Add(pk, item)

			pitems := items[pk]
			pitems = append(pitems, item)
			items[pk] = pitems
		}
	}

	for pk := range participants {
		skip := size
		expected := items[pk][skip:]
		cached, err := pec.Get(pk, skip)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected and cached not equal")
		}
	}
}
