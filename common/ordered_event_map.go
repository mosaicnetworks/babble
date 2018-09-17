package common

import "sort"

type OrderedMap map[int]interface{}

type Index struct {
	key   int
	value interface{}
}

type ByIndex []Index

func (a ByIndex) Len() int           { return len(a) }
func (a ByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByIndex) Less(i, j int) bool { return a[i].key < a[j].key }
func (a ByIndex) ToSlice() (res []interface{}) {
	for _, idx := range a {
		res = append(res, idx.key)
	}

	return
}

// Naive implementation, needs caching
func (o OrderedMap) ToOrderedSlice() []interface{} {
	idxs := ByIndex{}

	for k, v := range o {
		idxs = append(idxs, Index{k, v})
	}

	sort.Sort(idxs)

	return idxs.ToSlice()
}
