package common

import (
	"sort"
)

// Median gets the median number in a slice of numbers
func Median(input []int64) (median int64) {

	// Start by sorting a copy of the slice
	s := make([]int64, len(input))
	copy(s, input)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })

	// No math is needed if there are no numbers
	// For even numbers we add the two middle numbers and divide by two
	// For odd numbers we just use the middle number
	l := len(s)
	if l == 0 {
		return 0
	} else if l%2 == 0 {
		mid := l/2 - 1
		median = (s[mid] + s[mid+1]) / 2
	} else {
		median = s[l/2]
	}

	return median
}
