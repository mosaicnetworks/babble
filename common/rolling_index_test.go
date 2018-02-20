package common

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRollingIndex(t *testing.T) {
	size := 10
	testSize := 3 * size
	RollingIndex := NewRollingIndex(size)
	items := []string{}
	for i := 0; i < testSize; i++ {
		item := fmt.Sprintf("item%d", i)
		RollingIndex.Set(item, i)
		items = append(items, item)
	}
	cached, lastIndex := RollingIndex.GetLastWindow()

	expectedLastIndex := testSize - 1
	if lastIndex != expectedLastIndex {
		t.Fatalf("lastIndex should be %d, not %d", expectedLastIndex, lastIndex)
	}

	start := (testSize / (2 * size)) * (size)
	count := testSize - start

	for i := 0; i < count; i++ {
		if cached[i] != items[start+i] {
			t.Fatalf("cached[%d] should be %s, not %s", i, items[start+i], cached[i])
		}
	}

	err := RollingIndex.Set("ErrSkippedIndex", expectedLastIndex+2)
	if err == nil || !Is(err, SkippedIndex) {
		t.Fatalf("Should return ErrSkippedIndex")
	}

	_, err = RollingIndex.GetItem(9)
	if err == nil || !Is(err, TooLate) {
		t.Fatalf("Should return ErrTooLate")
	}

	var item interface{}

	indexes := []int{10, 17, 29}
	for _, i := range indexes {
		item, err = RollingIndex.GetItem(i)
		if err != nil {
			t.Fatalf("GetItem(%d) err: %v", i, err)
		}
		if !reflect.DeepEqual(item, items[i]) {
			t.Fatalf("GetItem error")
		}
	}

	_, err = RollingIndex.GetItem(lastIndex + 1)
	if err == nil || !Is(err, KeyNotFound) {
		t.Fatalf("Should return KeyNotFound")
	}

	//Test updating an item in place
	updateIndex := 26
	updateValue := "Updated Item"

	err = RollingIndex.Set(updateValue, updateIndex)
	if err != nil {
		t.Fatalf("SetItem(%d) err: %v", updateIndex, err)
	}
	item, err = RollingIndex.GetItem(updateIndex)
	if err != nil {
		t.Fatalf("GetItem(%d) err: %v", updateIndex, err)
	}
	if uv := item.(string); uv != updateValue {
		t.Fatalf("Updated item %d should be %s, not %s", updateIndex, updateValue, uv)
	}

}

func TestRollingIndexSkip(t *testing.T) {
	size := 10
	testSize := 25
	RollingIndex := NewRollingIndex(size)

	_, err := RollingIndex.Get(-1)
	if err != nil {
		t.Fatal(err)
	}

	items := []string{}
	for i := 0; i < testSize; i++ {
		item := fmt.Sprintf("item%d", i)
		RollingIndex.Set(item, i)
		items = append(items, item)
	}

	if _, err := RollingIndex.Get(0); err != nil && !Is(err, TooLate) {
		t.Fatalf("Skipping index 0 should return ErrTooLate")
	}

	skipIndex1 := 9
	expected1 := items[skipIndex1+1:]
	cached1, err := RollingIndex.Get(skipIndex1)
	if err != nil {
		t.Fatal(err)
	}
	convertedItems := []string{}
	for _, item := range cached1 {
		convertedItems = append(convertedItems, item.(string))
	}
	if !reflect.DeepEqual(expected1, convertedItems) {
		t.Fatalf("expected and cached not equal")
	}

	skipIndex2 := 15
	expected2 := items[skipIndex2+1:]
	cached2, err := RollingIndex.Get(skipIndex2)
	if err != nil {
		t.Fatal(err)
	}
	convertedItems = []string{}
	for _, item := range cached2 {
		convertedItems = append(convertedItems, item.(string))
	}
	if !reflect.DeepEqual(expected2, convertedItems) {
		t.Fatalf("expected and cached not equal")
	}

	skipIndex3 := 27
	expected3 := []string{}
	cached3, err := RollingIndex.Get(skipIndex3)
	if err != nil {
		t.Fatal(err)
	}
	convertedItems = []string{}
	for _, item := range cached3 {
		convertedItems = append(convertedItems, item.(string))
	}
	if !reflect.DeepEqual(expected3, convertedItems) {
		t.Fatalf("expected and cached not equal")
	}

}
