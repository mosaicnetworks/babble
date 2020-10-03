package common

import "testing"

func TestMedian(t *testing.T) {
	for _, c := range []struct {
		in  []int64
		out int64
	}{
		{[]int64{5, 3, 4, 2, 1}, 3},
		{[]int64{6, 3, 2, 4, 5, 1}, 3},
		{[]int64{1}, 1},
	} {
		got := Median(c.in)
		if got != c.out {
			t.Errorf("Median(%d) => %d != %d", c.in, got, c.out)
		}
	}
	m := Median([]int64{})
	if m != 0 {
		t.Errorf("Empty slice should have returned 0")
	}
}
