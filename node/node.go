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
	"fmt"
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

func (n *Node) Init() error {
	return n.core.Init()
}

func (n *Node) StartAsync() {
	go n.Listen()
}

func (n *Node) Listen() {
	for {
		select {
		case rpc := <-n.rpcCh:
			n.processRPC(rpc)
		case <-n.shutdownCh:
			return
		}
	}
}

func (n *Node) processRPC(rpc net.RPC) {
	switch cmd := rpc.Command.(type) {
	case *net.KnownRequest:
		n.processKnown(rpc, cmd)
	case *net.SyncRequest:
		n.processSync(rpc, cmd)
	default:
		n.logger.Printf("[ERR] node: Got unexpected command: %#v", rpc.Command)
		rpc.Respond(nil, fmt.Errorf("unexpected command"))
	}
}

func (n *Node) processKnown(rpc net.RPC, cmd *net.KnownRequest) {
	// Setup a response
	known := n.core.Known()
	resp := &net.KnownResponse{
		Known: known,
	}
	rpc.Respond(resp, nil)
}

func (n *Node) processSync(rpc net.RPC, cmd *net.SyncRequest) {
	success := true
	err := n.core.Sync(cmd.Head, cmd.Events, [][]byte{})
	if err != nil {
		success = false
	}
	resp := &net.SyncResponse{
		Success: success,
	}
	rpc.Respond(resp, err)
}
