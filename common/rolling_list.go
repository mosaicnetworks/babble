package common

import "errors"

var (
	ErrKeyNotFound = errors.New("not found")
	ErrTooLate     = errors.New("too late")
)

type RollingList struct {
	size  int
	tot   int
	items []interface{}
}

func NewRollingList(size int) *RollingList {
	return &RollingList{
		size:  size,
		items: make([]interface{}, 0, 2*size),
	}
}

func (r *RollingList) Get() (lastWindow []interface{}, tot int) {
	return r.items, r.tot
}

func (r *RollingList) GetItem(index int) (interface{}, error) {
	items := len(r.items)
	oldestCached := r.tot - items
	if index < oldestCached {
		return nil, ErrTooLate
	}
	findex := index - oldestCached
	if findex >= items {
		return nil, ErrKeyNotFound
	}
	return r.items[findex], nil
}

func (r *RollingList) Add(item interface{}) {
	if len(r.items) >= 2*r.size {
		r.Roll()
	}
	r.items = append(r.items, item)
	r.tot++
}

func (r *RollingList) Roll() {
	newList := make([]interface{}, 0, 2*r.size)
	newList = append(newList, r.items[r.size:]...)
	r.items = newList
}
