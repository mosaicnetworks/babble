package peers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

const (
	jsonPeerSetPath        = "peers.json"
	jsonGenesisPeerSetPath = "peers.genesis.json"
)

// JSONPeerSet is used to provide peer persistence on disk in the form
// of a JSON file.
type JSONPeerSet struct {
	l    sync.Mutex
	path string
}

// NewJSONPeerSet creates a new JSONPeerSet.
func NewJSONPeerSet(base string, isCurrent bool) *JSONPeerSet {
	var path string

	if isCurrent {
		path = filepath.Join(base, jsonPeerSetPath)
	} else {
		path = filepath.Join(base, jsonGenesisPeerSetPath)
	}

	store := &JSONPeerSet{
		path: path,
	}
	return store
}

//PeerSet creates a PeerSet from the JSONPeerSet
func (j *JSONPeerSet) PeerSet() (*PeerSet, error) {
	j.l.Lock()
	defer j.l.Unlock()

	// Read the file
	buf, err := ioutil.ReadFile(j.path)
	if err != nil {
		return nil, err
	}

	// Check for no peers
	if len(buf) == 0 {
		return nil, nil
	}

	// Decode the peers
	var peers []*Peer
	dec := json.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&peers); err != nil {
		return nil, err
	}

	cleansePeerSet(peers)

	return NewPeerSet(peers), nil
}

//NB this is safe because only value are altered, but no items are added / deleted
//so the slice header is unaffected
//This function standardises the peer string format passed into peerset. It matches
//the format Babble derives from a private key.
func cleansePeerSet(peers []*Peer) {
	for _, peer := range peers {
		peer.PubKeyHex = "0X" + strings.TrimPrefix((strings.ToUpper(peer.PubKeyHex)), "0X")
	}
}

//Write persists a PeerSet to a JSON file in path
func (j *JSONPeerSet) Write(peers []*Peer) error {
	j.l.Lock()
	defer j.l.Unlock()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peers); err != nil {
		return err
	}

	// Write out as JSON
	return ioutil.WriteFile(j.path, buf.Bytes(), 0755)
}
