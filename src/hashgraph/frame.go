package hashgraph

import (
	"bytes"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/ugorji/go/codec"
)

type Frame struct {
	Round  int //RoundReceived
	Peers  []*peers.Peer
	Roots  map[string]*Root
	Events []*Event //Event with RoundReceived = Round
}

//json encoding of Frame
func (f *Frame) Marshal() ([]byte, error) {
	b := new(bytes.Buffer)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	enc := codec.NewEncoder(b, jh)

	if err := enc.Encode(f); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (f *Frame) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	dec := codec.NewDecoder(b, jh)

	return dec.Decode(f)
}

func (f *Frame) Hash() ([]byte, error) {
	hashBytes, err := f.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}
