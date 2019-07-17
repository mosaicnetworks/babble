package hashgraph

import (
	"bytes"
	"sort"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/ugorji/go/codec"
)

// Frame ...
type Frame struct {
	Round    int //RoundReceived
	Peers    []*peers.Peer
	Roots    map[string]*Root
	Events   []*FrameEvent         //Events with RoundReceived = Round
	PeerSets map[int][]*peers.Peer //[round] => Peers
}

// SortedFrameEvents ...
func (f *Frame) SortedFrameEvents() []*FrameEvent {
	sorted := SortedFrameEvents{}
	for _, r := range f.Roots {
		sorted = append(sorted, r.Events...)
	}
	sorted = append(sorted, f.Events...)
	sort.Sort(sorted)
	return sorted
}

//Marshal - json encoding of Frame
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

// Unmarshal ...
func (f *Frame) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	dec := codec.NewDecoder(b, jh)

	if err := dec.Decode(f); err != nil {
		return err
	}

	return nil
}

// Hash ...
func (f *Frame) Hash() ([]byte, error) {
	hashBytes, err := f.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}
