package hashgraph

import (
	"bytes"
	"encoding/json"
	"math/big"
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

type RoundEvent struct {
	Witness bool
	Famous  Trilean
}

type RoundInfo struct {
	Events map[string]RoundEvent
	queued bool
}

func NewRoundInfo() *RoundInfo {
	return &RoundInfo{
		Events: make(map[string]RoundEvent),
	}
}

func (r *RoundInfo) AddEvent(x string, witness bool) {
	_, ok := r.Events[x]
	if !ok {
		r.Events[x] = RoundEvent{
			Witness: witness,
		}
	}
}

func (r *RoundInfo) SetFame(x string, f bool) {
	e, ok := r.Events[x]
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
	r.Events[x] = e
}

//return true if no witnesses' fame is left undefined
func (r *RoundInfo) WitnessesDecided() bool {
	for _, e := range r.Events {
		if e.Witness && e.Famous == Undefined {
			return false
		}
	}
	return true
}

//return witnesses
func (r *RoundInfo) Witnesses() []string {
	res := []string{}
	for x, e := range r.Events {
		if e.Witness {
			res = append(res, x)
		}
	}
	return res
}

//return famous witnesses
func (r *RoundInfo) FamousWitnesses() []string {
	res := []string{}
	for x, e := range r.Events {
		if e.Witness && e.Famous == True {
			res = append(res, x)
		}
	}
	return res
}

func (r *RoundInfo) IsDecided(witness string) bool {
	w, ok := r.Events[witness]
	return ok && w.Witness && w.Famous != Undefined
}

func (r *RoundInfo) PseudoRandomNumber() *big.Int {
	res := new(big.Int)
	for x, e := range r.Events {
		if e.Witness && e.Famous == True {
			s, _ := new(big.Int).SetString(x, 16)
			res = res.Xor(res, s)
		}
	}
	return res
}

func (r *RoundInfo) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (r *RoundInfo) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	return dec.Decode(r)
}
