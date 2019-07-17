package common

import (
	"fmt"
)

// RollingIndexMap ...
type RollingIndexMap struct {
	name    string
	size    int
	keys    []uint32
	mapping map[uint32]*RollingIndex
}

// NewRollingIndexMap ...
func NewRollingIndexMap(name string, size int) *RollingIndexMap {
	return &RollingIndexMap{
		name:    name,
		size:    size,
		keys:    []uint32{},
		mapping: make(map[uint32]*RollingIndex),
	}
}

// AddKey ...
func (rim *RollingIndexMap) AddKey(key uint32) error {
	if _, ok := rim.mapping[key]; ok {
		return NewStoreErr(rim.name, KeyAlreadyExists, fmt.Sprint(key))
	}
	rim.keys = append(rim.keys, key)
	rim.mapping[key] = NewRollingIndex(fmt.Sprintf("%s[%d]", rim.name, key), rim.size)
	return nil
}

//Get return key items with index > skip
func (rim *RollingIndexMap) Get(key uint32, skipIndex int) ([]interface{}, error) {
	items, ok := rim.mapping[key]
	if !ok {
		return nil, NewStoreErr(rim.name, KeyNotFound, fmt.Sprint(key))
	}

	cached, err := items.Get(skipIndex)
	if err != nil {
		return nil, err
	}

	return cached, nil
}

// GetItem ...
func (rim *RollingIndexMap) GetItem(key uint32, index int) (interface{}, error) {
	return rim.mapping[key].GetItem(index)
}

// GetLast ...
func (rim *RollingIndexMap) GetLast(key uint32) (interface{}, error) {
	pe, ok := rim.mapping[key]
	if !ok {
		return nil, NewStoreErr(rim.name, KeyNotFound, fmt.Sprint(key))
	}
	cached, _ := pe.GetLastWindow()
	if len(cached) == 0 {
		return "", NewStoreErr(rim.name, Empty, fmt.Sprint(key))
	}
	return cached[len(cached)-1], nil
}

// Set ...
func (rim *RollingIndexMap) Set(key uint32, item interface{}, index int) error {
	items, ok := rim.mapping[key]
	if !ok {
		items = NewRollingIndex(fmt.Sprintf("%s[%d]", rim.name, key), rim.size)
		rim.mapping[key] = items
	}
	return items.Set(item, index)
}

//Known returns [key] => lastKnownIndex
func (rim *RollingIndexMap) Known() map[uint32]int {
	known := make(map[uint32]int)
	for k, items := range rim.mapping {
		_, lastIndex := items.GetLastWindow()
		known[k] = lastIndex
	}
	return known
}
