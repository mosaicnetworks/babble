package hashgraph

import (
	"fmt"
	"reflect"
	"testing"

	cm "github.com/mosaicnetworks/babble/common"
)

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

			pec.Set(pk, item, i)

			pitems := items[pk]
			pitems = append(pitems, item)
			items[pk] = pitems
		}
	}

	// GET ITEM ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	for pk := range participants {

		index1 := 9
		_, err := pec.GetItem(pk, index1)
		if err == nil || !cm.Is(err, cm.TooLate) {
			t.Fatalf("Expected ErrTooLate")
		}

		index2 := 15
		expected2 := items[pk][index2]
		actual2, err := pec.GetItem(pk, index2)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected2, actual2) {
			t.Fatalf("expected and cached not equal")
		}

		index3 := 27
		expected3 := []string{}
		actual3, err := pec.Get(pk, index3)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected3, actual3) {
			t.Fatalf("expected and cached not equal")
		}
	}

	//KNOWN ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	known := pec.Known()
	for p, k := range known {
		expectedLastIndex := testSize - 1
		if k != expectedLastIndex {
			t.Errorf("Known[%d] should be %d, not %d", p, expectedLastIndex, k)
		}
	}

	//GET ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	for pk := range participants {
		if _, err := pec.Get(pk, 0); err != nil && !cm.Is(err, cm.TooLate) {
			t.Fatalf("Skipping 0 elements should return ErrTooLate")
		}

		skipIndex := 9
		expected := items[pk][skipIndex+1:]
		cached, err := pec.Get(pk, skipIndex)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected and cached not equal")
		}

		skipIndex2 := 15
		expected2 := items[pk][skipIndex2+1:]
		cached2, err := pec.Get(pk, skipIndex2)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected2, cached2) {
			t.Fatalf("expected and cached not equal")
		}

		skipIndex3 := 27
		expected3 := []string{}
		cached3, err := pec.Get(pk, skipIndex3)
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

			pec.Set(pk, item, i)

			pitems := items[pk]
			pitems = append(pitems, item)
			items[pk] = pitems
		}
	}

	for pk := range participants {
		expected := items[pk][size:]
		cached, err := pec.Get(pk, size-1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected (%#v) and cached (%#v) not equal", expected, cached)
		}
	}
}
