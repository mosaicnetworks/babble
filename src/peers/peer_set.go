package peers

import (
	"bytes"
	"encoding/json"
	"math"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
)

//PeerSet is a set of Peers forming a consensus network
type PeerSet struct {
	Peers    []*Peer          `json:"peers"`
	ByPubKey map[string]*Peer `json:"-"`
	ByID     map[uint32]*Peer `json:"-"`

	//cached values
	hash          []byte
	hex           string
	superMajority *int
	trustCount    *int
}

/* Constructors */

//NewPeerSet creates a new PeerSet from a list of Peers
func NewPeerSet(peers []*Peer) *PeerSet {
	peerSet := &PeerSet{
		ByPubKey: make(map[string]*Peer),
		ByID:     make(map[uint32]*Peer),
	}

	for _, peer := range peers {
		peerSet.ByPubKey[peer.PubKeyString()] = peer
		peerSet.ByID[peer.ID()] = peer
	}

	peerSet.Peers = peers

	return peerSet
}

//NewPeerSetFromPeerSliceBytes creates a new PeerSet from a peerSlice in Bytes format
func NewPeerSetFromPeerSliceBytes(peerSliceBytes []byte) (*PeerSet, error) {
	//Decode Peer slice
	peers := []*Peer{}

	b := bytes.NewBuffer(peerSliceBytes)
	dec := json.NewDecoder(b) //will read from b

	err := dec.Decode(&peers)
	if err != nil {
		return nil, err
	}
	//create new PeerSet
	return NewPeerSet(peers), nil
}

//WithNewPeer returns a new PeerSet with a list of peers including the new one.
func (peerSet *PeerSet) WithNewPeer(peer *Peer) *PeerSet {
	peers := peerSet.Peers

	//don't add it if it already exists
	if _, ok := peerSet.ByID[peer.ID()]; !ok {
		peers = append(peers, peer)
	}

	newPeerSet := NewPeerSet(peers)
	return newPeerSet
}

//WithRemovedPeer returns a new PeerSet with a list of peers excluding the
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
func (peerSet *PeerSet) PubKeys() []string {
	res := []string{}

	for _, peer := range peerSet.Peers {
		res = append(res, peer.PubKeyString())
	}

	return res
}

//IDs returns the PeerSet's slice of IDs
func (peerSet *PeerSet) IDs() []uint32 {
	res := []uint32{}

	for _, peer := range peerSet.Peers {
		res = append(res, peer.ID())
	}

	return res
}

/* Utilities */

//Len returns the number of Peers in the PeerSet
func (peerSet *PeerSet) Len() int {
	return len(peerSet.ByPubKey)
}

// Hash uniquely identifies a PeerSet. It is computed by hashing (SHA256) their
// public keys together, one by one.
func (peerSet *PeerSet) Hash() ([]byte, error) {
	if len(peerSet.hash) == 0 {
		hash := []byte{}
		for _, p := range peerSet.Peers {
			pk := p.PubKeyBytes()
			hash = crypto.SimpleHashFromTwoHashes(hash, pk)
		}
		peerSet.hash = hash
	}
	return peerSet.hash, nil
}

//Hex is the hexadecimal representation of Hash
func (peerSet *PeerSet) Hex() string {
	if len(peerSet.hex) == 0 {
		hash, _ := peerSet.Hash()
		peerSet.hex = common.EncodeToString(hash)
	}
	return peerSet.hex
}

//Marshal marshals the peerset
func (peerSet *PeerSet) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peerSet.Peers); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//SuperMajority return the number of peers that forms a strong majortiy (+2/3)
//in the PeerSet
func (peerSet *PeerSet) SuperMajority() int {
	if peerSet.superMajority == nil {
		val := 2*peerSet.Len()/3 + 1
		peerSet.superMajority = &val
	}
	return *peerSet.superMajority
}

//TrustCount calculates the Trust Count for a peerset
func (peerSet *PeerSet) TrustCount() int {
	if peerSet.trustCount == nil {
		val := 0
		if len(peerSet.Peers) > 1 {
			val = int(math.Ceil(float64(peerSet.Len()) / float64(3)))
		}
		peerSet.trustCount = &val
	}
	return *peerSet.trustCount
}

func (peerSet *PeerSet) clearCache() {
	peerSet.hash = []byte{}
	peerSet.hex = ""
	peerSet.superMajority = nil
}
