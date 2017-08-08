package common

import (
	"fmt"
	"testing"
)

func TestRollingWindow(t *testing.T) {
	size := 10
	testSize := 3 * size
	rollingList := NewRollingList(size)
	items := []string{}
	for i := 0; i < testSize; i++ {
		item := fmt.Sprintf("item%d", i)
		rollingList.Add(item)
		items = append(items, item)
	}
	cached, ts := rollingList.Get()

	if ts != testSize {
		t.Fatalf("tot should be %d, not %d", testSize, ts)
	}

	start := (testSize / (2 * size)) * (size)
	count := testSize - start

	for i := 0; i < count; i++ {
		if cached[i] != items[start+i] {
			t.Fatalf("cached[%d] should be %s, not %s", i, items[start+i], cached[i])
		}
	}
}
