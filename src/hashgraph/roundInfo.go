package hashgraph

import (
	"bytes"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/ugorji/go/codec"
)

type pendingRound struct {
	Index   int
	Decided bool
}

// RoundEvent ...
type RoundEvent struct {
	Witness bool
	Famous  common.Trilean
}

// RoundInfo ...
type RoundInfo struct {
	CreatedEvents  map[string]RoundEvent
	ReceivedEvents []string
	queued         bool
	decided        bool
}

// NewRoundInfo ...
func NewRoundInfo() *RoundInfo {
	return &RoundInfo{
		CreatedEvents:  make(map[string]RoundEvent),
		ReceivedEvents: []string{},
	}
}

// AddCreatedEvent ...
func (r *RoundInfo) AddCreatedEvent(x string, witness bool) {
	_, ok := r.CreatedEvents[x]
	if !ok {
		r.CreatedEvents[x] = RoundEvent{
			Witness: witness,
		}
	}
}

// AddReceivedEvent ...
func (r *RoundInfo) AddReceivedEvent(x string) {
	r.ReceivedEvents = append(r.ReceivedEvents, x)
}

// SetFame ...
func (r *RoundInfo) SetFame(x string, f bool) {
	e, ok := r.CreatedEvents[x]
	if !ok {
		e = RoundEvent{
			Witness: true,
		}
	}

	if f {
		e.Famous = common.True
	} else {
		e.Famous = common.False
	}

	r.CreatedEvents[x] = e
}

/*
WitnessesDecided returns true if a super-majority of witnesses are decided,
and there are no undecided witnesses. Our algorithm relies on the fact that a
witness that is not yet known when a super-majority of witnesses are already
decided, has no chance of ever being famous. Once a Round is decided it stays
decided, even if new witnesses are added after it was first decided.
*/
func (r *RoundInfo) WitnessesDecided(peerSet *peers.PeerSet) bool {
	//if the round was already decided, it stays decided no matter what.
	if r.decided {
		return true
	}

	c := 0
	for _, e := range r.CreatedEvents {
		if e.Witness && e.Famous != common.Undefined {
			c++
		} else if e.Witness && e.Famous == common.Undefined {
			return false
		}
	}

	r.decided = c >= peerSet.SuperMajority()

	return r.decided
}

//Witnesses return witnesses
func (r *RoundInfo) Witnesses() []string {
	res := []string{}
	for x, e := range r.CreatedEvents {
		if e.Witness {
			res = append(res, x)
		}
	}

	return res
}

//FamousWitnesses returns famous witnesses
func (r *RoundInfo) FamousWitnesses() []string {
	res := []string{}
	for x, e := range r.CreatedEvents {
		if e.Witness && e.Famous == common.True {
			res = append(res, x)
		}
	}
	return res
}

// IsDecided ...
func (r *RoundInfo) IsDecided(witness string) bool {
	w, ok := r.CreatedEvents[witness]
	return ok && w.Witness && w.Famous != common.Undefined
}

// Marshal ...
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

// Unmarshal ...
func (r *RoundInfo) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	dec := codec.NewDecoder(b, jh)

	return dec.Decode(r)
}

// IsQueued ...
func (r *RoundInfo) IsQueued() bool {
	return r.queued
}
