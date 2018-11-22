package hashgraph

import (
	"bytes"

	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/ugorji/go/codec"
)

type Trilean int

const (
	Undefined Trilean = iota
	True
	False
)

var trileans = []string{"Undefined", "True", "False"}

func (t Trilean) String() string {
	return trileans[t]
}

type pendingRound struct {
	Index   int
	Decided bool
}

type RoundEvent struct {
	Witness bool
	Famous  Trilean
}

type RoundInfo struct {
	CreatedEvents  map[string]RoundEvent
	ReceivedEvents []string
	queued         bool
}

func NewRoundInfo() *RoundInfo {
	return &RoundInfo{
		CreatedEvents:  make(map[string]RoundEvent),
		ReceivedEvents: []string{},
	}
}

func (r *RoundInfo) AddCreatedEvent(x string, witness bool) {
	_, ok := r.CreatedEvents[x]
	if !ok {
		r.CreatedEvents[x] = RoundEvent{
			Witness: witness,
		}
	}
}

func (r *RoundInfo) AddReceivedEvent(x string) {
	r.ReceivedEvents = append(r.ReceivedEvents, x)
}

func (r *RoundInfo) SetFame(x string, f bool) {
	e, ok := r.CreatedEvents[x]
	if !ok {
		e = RoundEvent{
			Witness: true,
		}
	}

	if f {
		e.Famous = True
	} else {
		e.Famous = False
	}

	r.CreatedEvents[x] = e
}

//return true if no witnesses' fame is left undefined
func (r *RoundInfo) WitnessesDecided(peerSet *peers.PeerSet) bool {
	c := 0
	for _, e := range r.CreatedEvents {
		if e.Witness && e.Famous != Undefined {
			c++
		}
	}
	return c >= peerSet.SuperMajority()
}

//return witnesses
func (r *RoundInfo) Witnesses() []string {
	res := []string{}
	for x, e := range r.CreatedEvents {
		if e.Witness {
			res = append(res, x)
		}
	}

	return res
}

//return famous witnesses
func (r *RoundInfo) FamousWitnesses() []string {
	res := []string{}
	for x, e := range r.CreatedEvents {
		if e.Witness && e.Famous == True {
			res = append(res, x)
		}
	}
	return res
}

func (r *RoundInfo) IsDecided(witness string) bool {
	w, ok := r.CreatedEvents[witness]
	return ok && w.Witness && w.Famous != Undefined
}

func (r *RoundInfo) Marshal() ([]byte, error) {
	b := new(bytes.Buffer)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	enc := codec.NewEncoder(b, jh)

	if err := enc.Encode(r); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (r *RoundInfo) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	dec := codec.NewDecoder(b, jh)

	return dec.Decode(r)
}

func (r *RoundInfo) IsQueued() bool {
	return r.queued
}
