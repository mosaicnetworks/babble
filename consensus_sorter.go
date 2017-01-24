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

import "math/big"

type ConsensusSorter struct {
	a     []Event
	r     map[int]RoundInfo
	cache map[int]*big.Int
}

func NewConsensusSorter(events []Event) ConsensusSorter {
	return ConsensusSorter{
		a:     events,
		r:     make(map[int]RoundInfo),
		cache: make(map[int]*big.Int),
	}
}

func (b ConsensusSorter) Len() int      { return len(b.a) }
func (b ConsensusSorter) Swap(i, j int) { b.a[i], b.a[j] = b.a[j], b.a[i] }
func (b ConsensusSorter) Less(i, j int) bool {
	if b.a[i].roundReceived != b.a[j].roundReceived {
		return b.a[i].roundReceived < b.a[j].roundReceived
	}

	if b.a[i].consensusTimestamp != b.a[j].consensusTimestamp {
		return b.a[i].consensusTimestamp.Sub(b.a[j].consensusTimestamp) < 0
	}

	w := b.GetPseudoRandomNumber(b.a[i].roundReceived)

	wsi := new(big.Int)
	wsi = wsi.Xor(b.a[i].S, w)
	wsj := new(big.Int)
	wsj = wsj.Xor(b.a[j].S, w)
	return wsi.Cmp(wsj) < 0
}
func (b ConsensusSorter) GetPseudoRandomNumber(round int) *big.Int {
	if ps, ok := b.cache[round]; ok {
		return ps
	}
	rd := b.r[round]
	ps := rd.PseudoRandomNumber()
	b.cache[round] = ps
	return ps
}
