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
package hashgraph

type Hashgraph struct {
	Events map[string]Event //hash => event
}

func NewHashgraph() Hashgraph {
	return Hashgraph{
		Events: make(map[string]Event),
	}
}

//returns true if y is an ancestor of x
func (h *Hashgraph) Ancestor(x, y string) bool {
	if x == "" {
		return false
	}
	ex, ok := h.Events[x]
	if !ok {
		return false
	}
	if x == y {
		return true
	}
	_, ok = h.Events[y]
	if !ok {
		return false
	}
	return h.Ancestor(ex.Body.Parents[0], y) || h.Ancestor(ex.Body.Parents[1], y)
}
