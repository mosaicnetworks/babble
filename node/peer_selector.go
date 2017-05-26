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
package node

import (
	"math/rand"
	"sort"
	"time"

	"bitbucket.org/mosaicnet/babble/net"
)

type PeerSelector interface {
	Peers() []net.Peer
	Update(peer string)
	Next() net.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//SMART

type peerWrapper struct {
	peer  net.Peer
	count int
	last  time.Time
}

type peerWrapperList []peerWrapper

func (p peerWrapperList) Len() int      { return len(p) }
func (p peerWrapperList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p peerWrapperList) Less(i, j int) bool {
	if p[i].count < p[j].count {
		return true
	}
	if p[i].count > p[j].count {
		return false
	}
	return p[i].last.Sub(p[j].last) < 0
}

type SmartPeerSelector struct {
	peers        []net.Peer
	peerWrappers map[string]*peerWrapper
	lastFrom     string
}

func NewSmartPeerSelector(participants []net.Peer, localAddr string) *SmartPeerSelector {
	_, peers := net.ExcludePeer(participants, localAddr)
	peerWrappers := make(map[string]*peerWrapper)
	for _, p := range peers {
		peerWrappers[p.NetAddr] = &peerWrapper{peer: p, count: 0}
	}
	return &SmartPeerSelector{
		peers:        peers,
		peerWrappers: peerWrappers,
	}
}

func (ps *SmartPeerSelector) Peers() []net.Peer {
	return ps.peers
}

func (ps *SmartPeerSelector) Update(peer string) {
	p, ok := ps.peerWrappers[peer]
	if ok {
		p.count++
		p.last = time.Now()
		ps.peerWrappers[peer] = p
	}
	ps.lastFrom = peer
}

func (ps *SmartPeerSelector) Next() net.Peer {
	if ps.lastFrom == "" {
		return ps.random()
	}
	ranked := ps.rank()
	return ranked[0].peer
}

func (ps *SmartPeerSelector) random() net.Peer {
	i := rand.Intn(len(ps.peers))
	peer := ps.peers[i]
	return peer
}

func (ps *SmartPeerSelector) rank() peerWrapperList {
	pl := make(peerWrapperList, len(ps.peers))
	i := 0
	for _, w := range ps.peerWrappers {
		pl[i] = *w
		i++
	}
	sort.Sort(pl)
	return pl
}

//+++++++++++++++++++++++++++++++++++++++
//RANDOM

type RandomPeerSelector struct {
	peers []net.Peer
	last  string
}

func NewRandomPeerSelector(participants []net.Peer, localAddr string) *RandomPeerSelector {
	_, peers := net.ExcludePeer(participants, localAddr)
	return &RandomPeerSelector{
		peers: peers,
	}
}

func (ps *RandomPeerSelector) Peers() []net.Peer {
	return ps.peers
}

func (ps *RandomPeerSelector) Update(peer string) {
	ps.last = peer
}

func (ps *RandomPeerSelector) Next() net.Peer {
	selectablePeers := ps.peers
	if len(selectablePeers) > 1 {
		_, selectablePeers = net.ExcludePeer(selectablePeers, ps.last)
	}
	i := rand.Intn(len(selectablePeers))
	peer := selectablePeers[i]
	return peer
}
