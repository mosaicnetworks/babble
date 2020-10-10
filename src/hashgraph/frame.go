package hashgraph

import (
	"bytes"
	"sort"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/ugorji/go/codec"
)

// Frame represents a section of the hashgraph.
type Frame struct {
	Round     int                   // RoundReceived
	Peers     []*peers.Peer         // the authoritative peer-set at Round
	Roots     map[string]*Root      // Roots on top of which Frame Events can be inserted
	Events    []*FrameEvent         // Events with RoundReceived = Round
	PeerSets  map[int][]*peers.Peer // full peer-set history ([round] => Peers)
	Timestamp int64                 // unix timestamp (median of round-received famous witnesses)
}

// SortedFrameEvents returns all the events in the Frame, including event is
// roots, sorted by LAMPORT timestamp
func (f *Frame) SortedFrameEvents() []*FrameEvent {
	sorted := SortedFrameEvents{}
	for _, r := range f.Roots {
		sorted = append(sorted, r.Events...)
	}
	sorted = append(sorted, f.Events...)
	sort.Sort(sorted)
	return sorted
}

// Marshal returns the JSON encoding of Frame.
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

// Unmarshal parses a JSON encoded Frame.
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

// Hash returns the SHA256 hash of the marshalled Frame.
func (f *Frame) Hash() ([]byte, error) {
	hashBytes, err := f.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}
