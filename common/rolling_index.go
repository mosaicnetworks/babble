package common

type RollingIndex struct {
	size      int
	lastIndex int
	items     []interface{}
}

func NewRollingIndex(size int) *RollingIndex {
	return &RollingIndex{
		size:      size,
		items:     make([]interface{}, 0, 2*size),
		lastIndex: -1,
	}
}

func (r *RollingIndex) GetLastWindow() (lastWindow []interface{}, lastIndex int) {
	return r.items, r.lastIndex
}

func (r *RollingIndex) Get(skipIndex int) ([]interface{}, error) {
	res := make([]interface{}, 0)

	if skipIndex > r.lastIndex {
		return res, nil
	}

	cachedItems := len(r.items)
	//assume there are no gaps between indexes
	oldestCachedIndex := r.lastIndex - cachedItems + 1
	if skipIndex+1 < oldestCachedIndex {
		return res, NewStoreErr(TooLate, string(SkippedIndex))
	}

	//index of 'skipped' in RollingIndex
	start := skipIndex - oldestCachedIndex + 1

	return r.items[start:], nil
}

func (r *RollingIndex) GetItem(index int) (interface{}, error) {
	items := len(r.items)
	oldestCached := r.lastIndex - items + 1
	if index < oldestCached {
		return nil, NewStoreErr(TooLate, string(index))
	}
	findex := index - oldestCached
	if findex >= items {
		return nil, NewStoreErr(KeyNotFound, string(index))
	}
	return r.items[findex], nil
}

func (r *RollingIndex) Add(item interface{}, index int) error {
	if index <= r.lastIndex {
		return NewStoreErr(PassedIndex, string(index))
	}
	if r.lastIndex >= 0 && index > r.lastIndex+1 {
		return NewStoreErr(SkippedIndex, string(index))
	}

	if len(r.items) >= 2*r.size {
		r.Roll()
	}
	r.items = append(r.items, item)
	r.lastIndex = index
	return nil
}

func (r *RollingIndex) Roll() {
	newList := make([]interface{}, 0, 2*r.size)
	newList = append(newList, r.items[r.size:]...)
	r.items = newList
}
