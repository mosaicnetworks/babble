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
	"crypto/ecdsa"
	"log"
	"os"
	"sync"

	"github.com/arrivets/go-swirlds/net"
)

type Node struct {
	core Core

	localAddr string
	logger    *log.Logger

	peers []net.Peer

	// RPC chan comes from the transport layer
	rpcCh <-chan net.RPC

	// Shutdown channel to exit, protected to prevent concurrent exits
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	// The transport layer we use
	trans net.Transport

	// Tracks running goroutines
	routinesGroup sync.WaitGroup
}

func NewNode(key *ecdsa.PrivateKey, participants []net.Peer, trans net.Transport) *Node {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	localAddr := trans.LocalAddr()

	participantPubs := []string{}
	for _, p := range participants {
		participantPubs = append(participantPubs, p.PubKeyHex)
	}
	core := NewCore(key, participantPubs)

	peers := net.ExcludePeer(participants, localAddr)

	node := &Node{
		core:       core,
		localAddr:  localAddr,
		logger:     logger,
		peers:      peers,
		rpcCh:      trans.Consumer(),
		shutdownCh: make(chan struct{}),
		trans:      trans,
	}
	return node
}
