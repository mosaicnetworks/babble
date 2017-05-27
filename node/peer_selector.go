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

	"bitbucket.org/mosaicnet/babble/net"
)

type PeerSelector interface {
	Peers() []net.Peer
	UpdateLast(peer string)
	UpdateDist(peer string, dist float64)
	Next() net.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//SMART

type SmartPeerSelector struct {
	peers    []net.Peer
	peersMap map[string]net.Peer

	last     string
	nextID   string
	nextDist float64
}

func NewSmartPeerSelector(participants []net.Peer, localAddr string) *SmartPeerSelector {
	_, peers := net.ExcludePeer(participants, localAddr)
	peersMap := make(map[string]net.Peer)
	for _, p := range peers {
		peersMap[p.NetAddr] = p
	}
	return &SmartPeerSelector{
		peers:    peers,
		peersMap: peersMap,
	}
}

func (ps *SmartPeerSelector) Peers() []net.Peer {
	return ps.peers
}

func (ps *SmartPeerSelector) UpdateLast(peer string) {
	ps.last = peer
}

func (ps *SmartPeerSelector) UpdateDist(peer string, dist float64) {
	if dist > ps.nextDist {
		ps.nextID = peer
		ps.nextDist = dist
	}
}

func (ps *SmartPeerSelector) Next() net.Peer {
	if ps.nextID != "" && ps.nextID != ps.last {
		return ps.peersMap[ps.nextID]
	}
	return ps.random()
}

func (ps *SmartPeerSelector) random() net.Peer {
	selectablePeers := ps.peers
	if len(selectablePeers) > 1 {
		_, selectablePeers = net.ExcludePeer(selectablePeers, ps.last)
	}
	i := rand.Intn(len(selectablePeers))
	peer := selectablePeers[i]
	return peer
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

func (ps *RandomPeerSelector) UpdateLast(peer string) {
	ps.last = peer
}

func (ps *RandomPeerSelector) UpdateDist(peer string, dist float64) {}

func (ps *RandomPeerSelector) Next() net.Peer {
	selectablePeers := ps.peers
	if len(selectablePeers) > 1 {
		_, selectablePeers = net.ExcludePeer(selectablePeers, ps.last)
	}
	i := rand.Intn(len(selectablePeers))
	peer := selectablePeers[i]
	return peer
}
