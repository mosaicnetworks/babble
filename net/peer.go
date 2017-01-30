/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package net

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	jsonPeerPath = "peers.json"
)

type Peer struct {
    NetAddr string
    PubKey  string
}

// PeerStore provides an interface for persistent storage and
// retrieval of peers.
type PeerStore interface {
	// Peers returns the list of known peers.
	Peers() ([]Peer, error)

	// SetPeers sets the list of known peers. This is invoked when a peer is
	// added or removed.
	SetPeers([]Peer) error
}

// StaticPeers is used to provide a static list of peers.
type StaticPeers struct {
	StaticPeers []Peer
	l           sync.Mutex
}

// Peers implements the PeerStore interface.
func (s *StaticPeers) Peers() ([]Peer, error) {
	s.l.Lock()
	peers := s.StaticPeers
	s.l.Unlock()
	return peers, nil
}

// SetPeers implements the PeerStore interface.
func (s *StaticPeers) SetPeers(p []Peer) error {
	s.l.Lock()
	s.StaticPeers = p
	s.l.Unlock()
	return nil
}

// JSONPeers is used to provide peer persistence on disk in the form
// of a JSON file. This allows human operators to manipulate the file.
type JSONPeers struct {
	l     sync.Mutex
	path  string
}

// NewJSONPeers creates a new JSONPeers store.
func NewJSONPeers(base string) *JSONPeers {
	path := filepath.Join(base, jsonPeerPath)
	store := &JSONPeers{
		path:  path,
	}
	return store
}

// Peers implements the PeerStore interface.
func (j *JSONPeers) Peers() ([]Peer, error) {
	j.l.Lock()
	defer j.l.Unlock()

	// Read the file
	buf, err := ioutil.ReadFile(j.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Check for no peers
	if len(buf) == 0 {
		return nil, nil
	}

	// Decode the peers
	var peerSet []Peer
	dec := json.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&peerSet); err != nil {
		return nil, err
	}

	return peerSet, nil
}

// SetPeers implements the PeerStore interface.
func (j *JSONPeers) SetPeers(peers []Peer) error {
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
