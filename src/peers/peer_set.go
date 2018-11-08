package peers

import (
	"fmt"
	"math"

	"github.com/mosaicnetworks/babble/src/crypto"
)

//XXX exclude peers should be in here

//PeerSet is a set of Peers forming a consensus network
type PeerSet struct {
	Peers    []*Peer          `json:"peers"`
	ByPubKey map[string]*Peer `json:"by_pub_key"`
	ByID     map[int]*Peer    `json:"by_id"`

	//cached values
	hash          []byte `json:"hash"`
	hex           string `json:"hex"`
	superMajority *int   `json:"super_majority"`
	trustCount    *int   `json:"trust_count"`
}

/* Constructors */

//NewPeerSet creates a new PeerSet from a list of Peers
func NewPeerSet(peers []*Peer) *PeerSet {
	peerSet := &PeerSet{
		ByPubKey: make(map[string]*Peer),
		ByID:     make(map[int]*Peer),
	}

	for _, peer := range peers {
		if peer.ID == 0 {
			peer.computeID()
		}

		peerSet.ByPubKey[peer.PubKeyHex] = peer
		peerSet.ByID[peer.ID] = peer
	}

	peerSet.Peers = peers

	return peerSet
}

//WithNewPeer returns a new PeerSet with a list of peers including the new one.
func (peerSet *PeerSet) WithNewPeer(peer *Peer) *PeerSet {
	peers := append(peerSet.Peers, peer)
	newPeerSet := NewPeerSet(peers)
	return newPeerSet
}

//WithRemovedPeer returns a new PeerSet with a list of peers exluding the
//provided one
func (peerSet *PeerSet) WithRemovedPeer(peer *Peer) *PeerSet {
	peers := []*Peer{}
	for _, p := range peerSet.Peers {
		if p.PubKeyHex != peer.PubKeyHex {
			peers = append(peers, p)
		}
	}
	newPeerSet := NewPeerSet(peers)
	return newPeerSet
}

/* ToSlice Methods */

//PubKeys returns the PeerSet's slice of public keys
func (c *PeerSet) PubKeys() []string {
	res := []string{}

	for _, peer := range c.Peers {
		res = append(res, peer.PubKeyHex)
	}

	return res
}

//IDs returns the PeerSet's slice of IDs
func (c *PeerSet) IDs() []int {
	res := []int{}

	for _, peer := range c.Peers {
		res = append(res, peer.ID)
	}

	return res
}

/* Utilities */

//Len returns the number of Peers in the PeerSet
func (c *PeerSet) Len() int {
	return len(c.ByPubKey)
}

//Hash uniquely identifies a PeerSet. It is computed by sorting the peers set
//by ID, and hashing (SHA256) their public keys together, one by one.
func (c *PeerSet) Hash() ([]byte, error) {
	if len(c.hash) == 0 {
		hash := []byte{}
		for _, p := range c.Peers {
			pk, _ := p.PubKeyBytes()
			hash = crypto.SimpleHashFromTwoHashes(hash, pk)
		}
		c.hash = hash
	}
	return c.hash, nil
}

//Hex is the hexadecimal representation of Hash
func (c *PeerSet) Hex() string {
	if len(c.hex) == 0 {
		hash, _ := c.Hash()
		c.hex = fmt.Sprintf("0x%X", hash)
	}
	return c.hex
}

//SuperMajority return the number of peers that forms a strong majortiy (+2/3)
//in the PeerSet
func (c *PeerSet) SuperMajority() int {
	if c.superMajority == nil {
		val := 2*c.Len()/3 + 1
		c.superMajority = &val
	}
	return *c.superMajority
}

func (c *PeerSet) TrustCount() int {
	if c.trustCount == nil {
		val := int(math.Ceil(float64(c.Len()) / float64(3)))
		c.trustCount = &val
	}
	return *c.trustCount
}

func (c *PeerSet) clearCache() {
	c.hash = []byte{}
	c.hex = ""
	c.superMajority = nil
}
