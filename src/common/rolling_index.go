package common

import "strconv"

// RollingIndex is a FIFO cache that evicts half of it's items when it reaches
// full capacity. It contains items in strict sequential order where index is a
// property of the items. It prevents skipping indexes when inserting new items.
type RollingIndex struct {
	name      string
	size      int
	lastIndex int
	items     []interface{}
}

// NewRollingIndex creates a new RollingIndex that contains up to size items.
func NewRollingIndex(name string, size int) *RollingIndex {
	return &RollingIndex{
		name:      name,
		size:      size,
		items:     make([]interface{}, 0, size),
		lastIndex: -1,
	}
}

// GetLastWindow returns all the items contained in the cache and the index of
// the last item.
func (r *RollingIndex) GetLastWindow() (lastWindow []interface{}, lastIndex int) {
	return r.items, r.lastIndex
}

// Get returns all the items with index greater than skipIndex or a TooLate
// error if some items have been evicted.
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

// GetItem retrieves an item by index. It returns a TooLate error if the item
// was evicted, or a KeyNotFound error if the item is not found.
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

// Set inserts an item and evicts the earlier half of the cache if it reached
// full capacity. It returns a SkippedIndex error if the item's index is greater
// than last index + 1. It allows setting items in place if the index
// corresponds to an existing item.
func (r *RollingIndex) Set(item interface{}, index int) error {
	// only allow setting items with index <= lastIndex + 1 so we may later
	// assume that there are no gaps between items
	if 0 <= r.lastIndex && index > r.lastIndex+1 {
		return NewStoreErr(r.name, SkippedIndex, strconv.Itoa(index))
	}

	// adding a new item
	if r.lastIndex < 0 || (index == r.lastIndex+1) {
		if len(r.items) >= r.size {
			r.roll()
		}
		r.items = append(r.items, item)
		r.lastIndex = index
		return nil
	}

	// replace an existing item. Make sure index is also greater or equal than
	// the oldest cached item's index
	cachedItems := len(r.items)
	oldestCachedIndex := r.lastIndex - cachedItems + 1

	if index < oldestCachedIndex {
		return NewStoreErr(r.name, TooLate, strconv.Itoa(index))
	}

	// replacing existing item
	position := index - oldestCachedIndex // position of 'index' in RollingIndex
	r.items[position] = item

	return nil
}

// roll evicts the earlier half of the cache and shifts the other half down.
func (r *RollingIndex) roll() {
	newList := make([]interface{}, 0, r.size)
	newList = append(newList, r.items[r.size/2:]...)
	r.items = newList
}
