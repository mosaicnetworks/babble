package peers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sync"
)

// JSONPeers is used to provide peer persistence on disk in the form
// of a JSON file. This allows human operators to manipulate the file.
type JSONPeers struct {
	l    sync.Mutex
	path string
}

// NewJSONPeers creates a new JSONPeers store.
func NewJSONPeers(base string) *JSONPeers {
	path := filepath.Join(base, jsonPeerPath)
	store := &JSONPeers{
		path: path,
	}
	return store
}

// Peers implements the PeerStore interface.
func (j *JSONPeers) Peers() (*Peers, error) {
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
	var peerSet []*Peer
	dec := json.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&peerSet); err != nil {
		return nil, err
	}

	return NewPeersFromSlice(peerSet), nil
}

// SetPeers implements the PeerStore interface.
func (j *JSONPeers) SetPeers(peers []*Peer) error {
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
