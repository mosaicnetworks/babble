package common

import (
	"fmt"
	"strconv"
)

type RollingIndexMap struct {
	name    string
	size    int
	keys    []int
	mapping map[int]*RollingIndex
}

func NewRollingIndexMap(name string, size int, keys []int) *RollingIndexMap {
	items := make(map[int]*RollingIndex)
	for _, key := range keys {
		items[key] = NewRollingIndex(fmt.Sprintf("%s[%d]", name, key), size)
	}
	return &RollingIndexMap{
		name:    name,
		size:    size,
		keys:    keys,
		mapping: items,
	}
}

//return key items with index > skip
func (rim *RollingIndexMap) Get(key int, skipIndex int) ([]interface{}, error) {
	items, ok := rim.mapping[key]
	if !ok {
		return nil, NewStoreErr(rim.name, KeyNotFound, strconv.Itoa(key))
	}

	cached, err := items.Get(skipIndex)
	if err != nil {
		return nil, err
	}

	return cached, nil
}

func (rim *RollingIndexMap) GetItem(key int, index int) (interface{}, error) {
	return rim.mapping[key].GetItem(index)
}

func (rim *RollingIndexMap) GetLast(key int) (interface{}, error) {
	pe, ok := rim.mapping[key]
	if !ok {
		return nil, NewStoreErr(rim.name, KeyNotFound, strconv.Itoa(key))
	}
	cached, _ := pe.GetLastWindow()
	if len(cached) == 0 {
		return "", NewStoreErr(rim.name, Empty, "")
	}
	return cached[len(cached)-1], nil
}

func (rim *RollingIndexMap) Set(key int, item interface{}, index int) error {
	items, ok := rim.mapping[key]
	if !ok {
		items = NewRollingIndex(fmt.Sprintf("%s[%d]", rim.name, key), rim.size)
		rim.mapping[key] = items
	}
	return items.Set(item, index)
}

//returns [key] => lastKnownIndex
func (rim *RollingIndexMap) Known() map[int]int {
	known := make(map[int]int)
	for k, items := range rim.mapping {
		_, lastIndex := items.GetLastWindow()
		known[k] = lastIndex
	}
	return known
}

func (rim *RollingIndexMap) Reset() error {
	items := make(map[int]*RollingIndex)
	for _, key := range rim.keys {
		items[key] = NewRollingIndex(fmt.Sprintf("%s[%d]", rim.name, key), rim.size)
	}
	rim.mapping = items
	return nil
}
