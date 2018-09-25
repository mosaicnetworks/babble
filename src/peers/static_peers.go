package peers

import "sync"

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
