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

type RoundInfo struct {
	Witnesses map[string]Trilean //witness => famous
	Events    []string
}

func (r *RoundInfo) AddEvent(x string, witness bool) {
	r.Events = append(r.Events, x)
	if witness {
		r.AddWitness(x)
	}
}

func (r *RoundInfo) AddWitness(x string) {
	if r.Witnesses == nil {
		r.Witnesses = make(map[string]Trilean)
	}
	r.Witnesses[x] = Undefined
}

func (r *RoundInfo) SetFame(x string, f bool) {
	if r.Witnesses == nil {
		r.Witnesses = make(map[string]Trilean)
	}
	if f {
		r.Witnesses[x] = True
	} else {
		r.Witnesses[x] = False
	}

}

//return true if no witnesses' fame is left undefined
func (r *RoundInfo) WitnessesDecided() bool {
	for _, f := range r.Witnesses {
		if f == Undefined {
			return false
		}
	}
	return true
}

//return famous witnesses
func (r *RoundInfo) FamousWitnesses() []string {
	res := []string{}
	for w, f := range r.Witnesses {
		if f == True {
			res = append(res, w)
		}
	}
	return res
}

func (r *RoundInfo) PseudoRandomNumber() *big.Int {
	res := new(big.Int)
	for w, t := range r.Witnesses {
		if t == True {
			s, _ := new(big.Int).SetString(w, 16)
			res = res.Xor(res, s)
		}
	}
	return res
}
