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

import (
	"fmt"
	"reflect"
)

type Hashgraph struct {
	Participants []string         //particant public keys
	Events       map[string]Event //hash => event
}

func NewHashgraph(participants []string) Hashgraph {
	return Hashgraph{
		Participants: participants,
		Events:       make(map[string]Event),
	}
}

func (h *Hashgraph) SuperMajority() int {
	return 2*len(h.Participants)/3 + 1
}

//true if y is an ancestor of x
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

//true if y is a self-ancestor of x
func (h *Hashgraph) SelfAncestor(x, y string) bool {
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
	return h.SelfAncestor(ex.Body.Parents[0], y)
}

//true if x detects a fork under y. also returns the hash of the event which
//is not a self-ancestor of y and caused the fork
func (h *Hashgraph) DetectFork(x, y string) bool {
	if x == "" || y == "" {
		return false
	}
	_, ok := h.Events[x]
	if !ok {
		return false
	}
	ey, ok := h.Events[y]
	if !ok {
		return false
	}

	//filter events satisfying the following criteria:
	//- same creator as y
	//- ancestors of x
	filteredEvents := []Event{}
	for hash, event := range h.Events {
		if reflect.DeepEqual(event.Body.Creator, ey.Body.Creator) &&
			h.Ancestor(x, hash) {
			filteredEvents = append(filteredEvents, event)
		}
	}
	for i := 0; i < len(filteredEvents)-1; i++ {
		a := filteredEvents[i].Hex()
		for j := i + 1; j < len(filteredEvents); j++ {
			b := filteredEvents[j].Hex()
			if !((h.SelfAncestor(a, b)) || h.SelfAncestor(b, a)) {
				return true
			}
		}
	}
	return false
}

//true if x sees y
func (h *Hashgraph) See(x, y string) bool {
	return h.Ancestor(x, y) && !h.DetectFork(x, y)
}

//participants in x's ancestry that see y
func (h *Hashgraph) MapSentinels(x, y string, sentinels map[string]bool) {
	if x == "" {
		return
	}
	ex, ok := h.Events[x]
	if !ok {
		return
	}
	if !h.See(x, y) {
		return
	}
	sentinels[fmt.Sprintf("0x%X", ex.Body.Creator)] = true
	if x == y {
		return
	}

	h.MapSentinels(ex.Body.Parents[0], y, sentinels)
	h.MapSentinels(ex.Body.Parents[1], y, sentinels)
}

//true if x strongly sees y
func (h *Hashgraph) StronglySee(x, y string) (bool, int) {
	sentinels := make(map[string]bool)
	for i := 0; i < len(h.Participants); i++ {
		sentinels[h.Participants[i]] = false
	}

	h.MapSentinels(x, y, sentinels)

	c := 0
	for _, v := range sentinels {
		if v {
			c++
		}
	}
	return c >= h.SuperMajority(), c
}
