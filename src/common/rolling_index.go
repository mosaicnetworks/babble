package common

import "strconv"

// RollingIndex ...
type RollingIndex struct {
	name      string
	size      int
	lastIndex int
	items     []interface{}
}

// NewRollingIndex ...
func NewRollingIndex(name string, size int) *RollingIndex {
	return &RollingIndex{
		name:      name,
		size:      size,
		items:     make([]interface{}, 0, 2*size),
		lastIndex: -1,
	}
}

// GetLastWindow ...
func (r *RollingIndex) GetLastWindow() (lastWindow []interface{}, lastIndex int) {
	return r.items, r.lastIndex
}

// Get ...
func (r *RollingIndex) Get(skipIndex int) ([]interface{}, error) {
	res := make([]interface{}, 0)

	if skipIndex > r.lastIndex {
		return res, nil
	}

	cachedItems := len(r.items)
	//assume there are no gaps between indexes
	oldestCachedIndex := r.lastIndex - cachedItems + 1
	if skipIndex+1 < oldestCachedIndex {
		return res, NewStoreErr(r.name, TooLate, strconv.Itoa(skipIndex))
	}

	//index of 'skipped' in RollingIndex
	start := skipIndex - oldestCachedIndex + 1

	return r.items[start:], nil
}

// GetItem ...
func (r *RollingIndex) GetItem(index int) (interface{}, error) {
	items := len(r.items)
	oldestCached := r.lastIndex - items + 1
	if index < oldestCached {
		return nil, NewStoreErr(r.name, TooLate, strconv.Itoa(index))
	}
	findex := index - oldestCached
	if findex >= items {
		return nil, NewStoreErr(r.name, KeyNotFound, strconv.Itoa(index))
	}
	return r.items[findex], nil
}

// Set ...
func (r *RollingIndex) Set(item interface{}, index int) error {
	//only allow to setting items with index <= lastIndex + 1 so we may assume
	//there are no gaps between items
	if 0 <= r.lastIndex && index > r.lastIndex+1 {
		return NewStoreErr(r.name, SkippedIndex, strconv.Itoa(index))
	}

	//adding a new item
	if r.lastIndex < 0 || (index == r.lastIndex+1) {
		if len(r.items) >= 2*r.size {
			r.Roll()
		}
		r.items = append(r.items, item)
		r.lastIndex = index
		return nil
	}

	//replace an existing item. Make sure index is also greater or equal than
	//the oldest cached item's index
	cachedItems := len(r.items)
	oldestCachedIndex := r.lastIndex - cachedItems + 1

	if index < oldestCachedIndex {
		return NewStoreErr(r.name, TooLate, strconv.Itoa(index))
	}

	//replacing existing item
	position := index - oldestCachedIndex //position of 'index' in RollingIndex
	r.items[position] = item

	return nil
}

// Roll ...
func (r *RollingIndex) Roll() {
	newList := make([]interface{}, 0, 2*r.size)
	newList = append(newList, r.items[r.size:]...)
	r.items = newList
}
