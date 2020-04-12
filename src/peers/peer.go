package peers

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
)

// Peer is a struct that holds Peer data.
type Peer struct {
	// NetAddr is the IP:PORT of the Babble node. Not necessary with WebRTC.
	NetAddr string
	// PubKeyHex is the hexadecimal representation of the peer's public key.
	PubKeyHex string
	// Moniker is an optional friendly name for the peer. It does not need to be
	// unique.
	Moniker string

	id uint32
}

// NewPeer instantiates a Peer.
func NewPeer(pubKeyHex, netAddr, moniker string) *Peer {
	peer := &Peer{
		PubKeyHex: pubKeyHex,
		NetAddr:   netAddr,
		Moniker:   moniker,
	}
	return peer
}

// ID returns an ID for the peer, calculated from the public key.
func (p *Peer) ID() uint32 {
	if p.id == 0 {
		pubKeyBytes := p.PubKeyBytes()
		p.id = keys.PublicKeyID(pubKeyBytes)
	}
	return p.id
}

// PubKeyString returns the upper-case version of PubKeyHex. It is used for
// indexing in maps with string keys.
func (p *Peer) PubKeyString() string {
	return strings.ToUpper(p.PubKeyHex)
}

// PubKeyBytes returns the byte slice representation of the Peer's public key.
func (p *Peer) PubKeyBytes() []byte {
	res, _ := common.DecodeFromString(p.PubKeyHex)
	return res
}

// Marshal marshals the Peer object. Note that this excludes the id field,
// forcing consumers to recalculate it.
func (p *Peer) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)

	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Unmarshal unmarshals a byte slice into a Peer object.
func (p *Peer) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)

	dec := json.NewDecoder(b) //will read from b

	if err := dec.Decode(p); err != nil {
		return err
	}

	return nil
}

// ExcludePeer is used to exclude a single peer from a list of peers.
func ExcludePeer(peers []*Peer, peer uint32) (int, []*Peer) {
	index := -1
	otherPeers := make([]*Peer, 0, len(peers))
	for i, p := range peers {
		if p.ID() != peer {
			otherPeers = append(otherPeers, p)
		} else {
			index = i
		}
	}
	return index, otherPeers
}
