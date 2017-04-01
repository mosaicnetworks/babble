/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package common

type RollingList struct {
	size  int
	tot   int
	items []interface{}
}

func NewRollingList(size int) *RollingList {
	return &RollingList{
		size:  size,
		items: make([]interface{}, 0, size),
	}
}

func (r *RollingList) Get() (lastWindow []interface{}, tot int) {
	return r.items, r.tot
}

func (r *RollingList) Add(item interface{}) {
	if len(r.items) >= r.size {
		r.Roll()
	}
	r.items = append(r.items, item)
	r.tot++
}

func (r *RollingList) Roll() {
	newList := make([]interface{}, 0, r.size)
	r.items = newList
}
