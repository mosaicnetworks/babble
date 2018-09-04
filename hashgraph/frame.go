package hashgraph

import (
	"bytes"
	"encoding/json"

	"github.com/mosaicnetworks/babble/crypto"
)

type Frame struct {
	Round  int     //RoundReceived
	Roots  []Root  // [participant ID] => Root
	Events []Event //Event with RoundReceived = Round
}

//json encoding of Frame
func (f *Frame) Marshal() ([]byte, error) {

	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(f); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (f *Frame) Unmarshal(data []byte) error {

	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	return dec.Decode(f)
}

func (f *Frame) Hash() ([]byte, error) {
	hashBytes, err := f.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}
