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

//++++++++++++++++++++++++++++++++++++++++++++++++
//EVENT CACHE
type EventCache struct {
	size   int
	count  int
	tot    int
	events map[string]Event
	buffer []string
	index  []string
}

func NewEventCache(size int) *EventCache {
	return &EventCache{
		size:   size,
		events: make(map[string]Event),
		buffer: make([]string, 0, size),
		index:  make([]string, 0, size),
	}
}

func (ec *EventCache) Get(hash string) (Event, bool) {
	e, ok := ec.events[hash]
	return e, ok
}

func (ec *EventCache) Set(event Event) {
	key := event.Hex()

	_, ok := ec.Get(key)
	if !ok {
		if ec.count >= ec.size {
			ec.Roll()
		}
		ec.tot++
	}

	ec.events[key] = event
	ec.index = append(ec.index, key)
	ec.count++
}

func (ec *EventCache) Roll() {
	//delete items from map
	for _, h := range ec.buffer {
		delete(ec.events, h)
	}
	//reset index
	ec.buffer = ec.index
	ec.index = make([]string, 0, ec.size)
	ec.count = 0
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//ROUND CACHE
type RoundCache struct {
	size   int
	count  int
	tot    int
	rounds map[int]RoundInfo
	buffer []int
	index  []int
}

func NewRoundCache(size int) *RoundCache {
	return &RoundCache{
		size:   size,
		rounds: make(map[int]RoundInfo),
		buffer: make([]int, 0, size),
		index:  make([]int, 0, size),
	}
}

func (rc *RoundCache) Get(round int) (RoundInfo, bool) {
	r, ok := rc.rounds[round]
	return r, ok
}

func (rc *RoundCache) Set(num int, round RoundInfo) {
	_, ok := rc.Get(num)
	if !ok {
		if rc.count >= rc.size {
			rc.Roll()
		}
		rc.tot++
	}
	rc.rounds[num] = round
	rc.index = append(rc.index, num)
	rc.count++
}

func (rc *RoundCache) Roll() {
	//delete items from map
	for _, r := range rc.buffer {
		delete(rc.rounds, r)
	}
	//reset index
	rc.buffer = rc.index
	rc.index = make([]int, 0, rc.size)
	rc.count = 0
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//STRING LIST CACHE
type StringListCache struct {
	size  int
	tot   int
	items []string
}

func NewStringListCache(size int) *StringListCache {
	return &StringListCache{
		size:  size,
		items: make([]string, 0, size),
	}
}

func (slc *StringListCache) Get() (lastWindow []string, tot int) {
	return slc.items, slc.tot
}

func (slc *StringListCache) Set(item string) {
	if len(slc.items) >= slc.size {
		slc.Roll()
	}
	slc.items = append(slc.items, item)
	slc.tot++
}

func (slc *StringListCache) Roll() {
	newList := make([]string, 0, slc.size)
	slc.items = newList
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//PARTICIPANT EVENTS CACHE
type ParticipantEventsCache struct {
	size              int
	participantEvents map[string]*StringListCache
}

func NewParticipantEventsCache(size int, participants []string) *ParticipantEventsCache {
	items := make(map[string]*StringListCache)
	for _, p := range participants {
		items[p] = NewStringListCache(size)
	}
	return &ParticipantEventsCache{
		size:              size,
		participantEvents: items,
	}
}

func (pec *ParticipantEventsCache) Get(participant string, skip int) ([]string, error) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		return []string{}, ErrKeyNotFound
	}

	if len(pe.items) == 0 {
		return pe.items, nil
	}

	lastCached := pe.tot - len(pe.items)
	if lastCached > skip {
		//XXX TODO
		//LOAD REST FROM FILE
		return []string{}, ErrTooLate
	}

	start := skip % pec.size
	if start >= len(pe.items) {
		return []string{}, nil
	}

	return pe.items[start:], nil
}

func (pec *ParticipantEventsCache) GetLast(participant string) (string, error) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		return "", ErrKeyNotFound
	}
	if len(pe.items) == 0 {
		return "", nil
	}
	return pe.items[len(pe.items)-1], nil
}

func (pec *ParticipantEventsCache) Set(participant string, hash string) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		pe = NewStringListCache(pec.size)
		pec.participantEvents[participant] = pe
	}
	pe.Set(hash)
}

func (pec *ParticipantEventsCache) Known() map[string]int {
	known := make(map[string]int)
	for p, evs := range pec.participantEvents {
		known[p] = evs.tot
	}
	return known
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//[KEY]=>BOOL CACHE

type Key struct {
	x string
	y string
}

type KeyBoolMapCache struct {
	size   int
	count  int
	tot    int
	items  map[Key]bool
	buffer []Key
	index  []Key
}

func NewKeyBoolMapCache(size int) *KeyBoolMapCache {
	return &KeyBoolMapCache{
		size:   size,
		items:  make(map[Key]bool),
		buffer: make([]Key, 0, size),
		index:  make([]Key, 0, size),
	}
}

func (c *KeyBoolMapCache) Get(key Key) (bool, bool) {
	e, ok := c.items[key]
	return e, ok
}

func (c *KeyBoolMapCache) Set(key Key, val bool) {
	_, ok := c.Get(key)
	if !ok {
		if c.count >= c.size {
			c.Roll()
		}
		c.tot++
	}

	c.items[key] = val
	c.index = append(c.index, key)
	c.count++
}

func (c *KeyBoolMapCache) Roll() {
	//delete items from map
	for _, h := range c.buffer {
		delete(c.items, h)
	}
	//reset index
	c.buffer = c.index
	c.index = make([]Key, 0, c.size)
	c.count = 0
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//[KEY]=>STRING CACHE

type KeyStringMapCache struct {
	size   int
	count  int
	tot    int
	items  map[Key]string
	buffer []Key
	index  []Key
}

func NewKeyStringMapCache(size int) *KeyStringMapCache {
	return &KeyStringMapCache{
		size:   size,
		items:  make(map[Key]string),
		buffer: make([]Key, 0, size),
		index:  make([]Key, 0, size),
	}
}

func (c *KeyStringMapCache) Get(key Key) (string, bool) {
	e, ok := c.items[key]
	return e, ok
}

func (c *KeyStringMapCache) Set(key Key, val string) {
	_, ok := c.Get(key)
	if !ok {
		if c.count >= c.size {
			c.Roll()
		}
		c.tot++
	}

	c.items[key] = val
	c.index = append(c.index, key)
	c.count++
}

func (c *KeyStringMapCache) Roll() {
	//delete items from map
	for _, h := range c.buffer {
		delete(c.items, h)
	}
	//reset index
	c.buffer = c.index
	c.index = make([]Key, 0, c.size)
	c.count = 0
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//[KEY]=>STRING CACHE

type StringIntMapCache struct {
	size   int
	count  int
	tot    int
	items  map[string]int
	buffer []string
	index  []string
}

func NewStringIntMapCache(size int) *StringIntMapCache {
	return &StringIntMapCache{
		size:   size,
		items:  make(map[string]int),
		buffer: make([]string, 0, size),
		index:  make([]string, 0, size),
	}
}

func (c *StringIntMapCache) Get(key string) (int, bool) {
	e, ok := c.items[key]
	return e, ok
}

func (c *StringIntMapCache) Set(key string, val int) {
	_, ok := c.Get(key)
	if !ok {
		if c.count >= c.size {
			c.Roll()
		}
		c.tot++
	}

	c.items[key] = val
	c.index = append(c.index, key)
	c.count++
}

func (c *StringIntMapCache) Roll() {
	//delete items from map
	for _, h := range c.buffer {
		delete(c.items, h)
	}
	//reset index
	c.buffer = c.index
	c.index = make([]string, 0, c.size)
	c.count = 0
}
