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

// JSONPeerSet is used to provide peer persistence on disk in the form of a JSON
// file.
type JSONPeerSet struct {
	l    sync.Mutex
	path string
}

// NewJSONPeerSet creates a new JSONPeerSet with reference to a base directory
// where the JSON files reside.
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

// PeerSet parses the underlying JSON file and returns the corresponding
// PeerSet.
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

// cleansePeerSet standardises the public key strings to match the format Babble
// derives from a private key.
func cleansePeerSet(peers []*Peer) {
	for _, peer := range peers {
		peer.PubKeyHex = "0X" + strings.TrimPrefix((strings.ToUpper(peer.PubKeyHex)), "0X")
	}
}

// Write persists a PeerSet to a JSON file.
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
