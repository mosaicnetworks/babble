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
	irr, jrr := -1, -1
	if b.a[i].roundReceived != nil {
		irr = *b.a[i].roundReceived
	}
	if b.a[j].roundReceived != nil {
		jrr = *b.a[j].roundReceived
	}
	if irr != jrr {
		return irr < jrr
	}

	if !b.a[i].consensusTimestamp.Equal(b.a[j].consensusTimestamp) {
		return b.a[i].consensusTimestamp.Before(b.a[j].consensusTimestamp)
	}

	w := b.GetPseudoRandomNumber(*b.a[i].roundReceived)
	wsi := new(big.Int)
	wsi = wsi.Xor(&b.a[i].S, w)
	wsj := new(big.Int)
	wsj = wsj.Xor(&b.a[j].S, w)
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
