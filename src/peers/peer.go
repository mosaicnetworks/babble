package peers

import (
	"encoding/hex"

	"github.com/mosaicnetworks/babble/src/common"
)

type Peer struct {
	ID        uint32 `json:"-"`
	NetAddr   string
	PubKeyHex string
}

func NewPeer(pubKeyHex, netAddr string) *Peer {
	peer := &Peer{
		PubKeyHex: pubKeyHex,
		NetAddr:   netAddr,
	}

	peer.ComputeID()

	return peer
}

func (p *Peer) PubKeyBytes() ([]byte, error) {
	return hex.DecodeString(p.PubKeyHex[2:])
}

func (p *Peer) ComputeID() error {
	// TODO: Use the decoded bytes from hex
	pubKey, err := p.PubKeyBytes()

	if err != nil {
		return err
	}

	p.ID = common.Hash32(pubKey)

	return nil
}

// ExcludePeer is used to exclude a single peer from a list of peers.
func ExcludePeer(peers []*Peer, peer uint32) (int, []*Peer) {
	index := -1
	otherPeers := make([]*Peer, 0, len(peers))
	for i, p := range peers {
		if p.ID != peer {
			otherPeers = append(otherPeers, p)
		} else {
			index = i
		}
	}
	return index, otherPeers
}
