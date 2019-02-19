package peers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/mosaicnetworks/babble/src/common"
)

type Peer struct {
	NetAddr   string
	PubKeyHex string
	Moniker   string

	id uint32
}

func NewPeer(pubKeyHex, netAddr, moniker string) *Peer {
	peer := &Peer{
		PubKeyHex: pubKeyHex,
		NetAddr:   netAddr,
		Moniker:   moniker,
	}
	return peer
}

func (p *Peer) ID() uint32 {
	if p.id == 0 {
		pubKeyBytes := p.PubKeyBytes()
		p.id = common.Hash32(pubKeyBytes)
	}
	return p.id
}

func (p *Peer) PubKeyBytes() []byte {
	res, _ := hex.DecodeString(p.PubKeyHex[2:])
	return res
}

//json encoding excludes the ID field
func (p *Peer) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)

	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

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
