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
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/arrivets/go-swirlds/hashgraph"
	"github.com/arrivets/go-swirlds/net"
)

type Node struct {
	conf *Config

	core *Core

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
}

func NewNode(conf *Config, key *ecdsa.PrivateKey, participants []net.Peer, trans net.Transport) Node {
	var logger *log.Logger
	if conf.Logger != nil {
		logger = conf.Logger
	} else {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	localAddr := trans.LocalAddr()

	participantPubs := []string{}
	for _, p := range participants {
		participantPubs = append(participantPubs, p.PubKeyHex)
	}
	core := NewCore(key, participantPubs)

	peers := net.ExcludePeer(participants, localAddr)

	node := Node{
		conf:       conf,
		core:       &core,
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
	peerAddresses := []string{}
	for _, p := range n.peers {
		peerAddresses = append(peerAddresses, p.NetAddr)
	}
	n.logger.Printf("[DEBUG] node: %s peers: %#v", n.localAddr, peerAddresses)
	return n.core.Init()
}

func (n *Node) RunAsync(gossip bool) {
	n.logger.Printf("[DEBUG] node: %s runasync", n.localAddr)
	go n.run(gossip)
}

func (n *Node) run(gossip bool) {
	heartbeatTimer := randomTimeout(n.conf.HeartbeatTimeout)
	for {
		select {
		case rpc := <-n.rpcCh:
			n.logger.Printf("[DEBUG] node: %s processing RPC", n.localAddr)
			n.processRPC(rpc)
		case <-heartbeatTimer:
			if gossip {
				n.logger.Printf("[DEBUG] node: %s time to Gossip!", n.localAddr)
				n.gossip()
			}
			// Restart the heartbeat timer
			heartbeatTimer = randomTimeout(n.conf.HeartbeatTimeout)
		case <-n.shutdownCh:
			return
		default:
		}
	}
}

func (n *Node) processRPC(rpc net.RPC) {
	switch cmd := rpc.Command.(type) {
	case *net.KnownRequest:
		n.logger.Printf("[DEBUG] node: %s processing known", n.localAddr)
		n.processKnown(rpc, cmd)
	case *net.SyncRequest:
		n.logger.Printf("[DEBUG] node: %s processing sync", n.localAddr)
		n.processSync(rpc, cmd)
	default:
		n.logger.Printf("[ERR] node: Got unexpected command: %#v", rpc.Command)
		rpc.Respond(nil, fmt.Errorf("unexpected command"))
	}
}

func (n *Node) processKnown(rpc net.RPC, cmd *net.KnownRequest) {
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
	} else {
		n.core.RunConsensus()
	}
	resp := &net.SyncResponse{
		Success: success,
	}
	rpc.Respond(resp, err)
}

func (n *Node) gossip() {
	i := rand.Intn(len(n.peers))
	peer := n.peers[i]

	known, err := n.requestKnown(peer.NetAddr)
	if err != nil {
		n.logger.Printf("[ERR] node: Getting peer Known: %s", err)
	} else {
		head, diff := n.core.Diff(known)
		if err := n.requestSync(peer.NetAddr, head, diff); err != nil {
			n.logger.Printf("[ERR] node: Triggering peer Sync: %s", err)
		}
	}
}

func (n *Node) requestKnown(target string) (map[string]int, error) {
	args := net.KnownRequest{
		From: n.localAddr,
	}
	var out net.KnownResponse
	if err := n.trans.RequestKnown(target, &args, &out); err != nil {
		return nil, err
	}
	n.logger.Printf("[DEBUG] node: %s replied to Known with %#v", target, out.Known)
	return out.Known, nil
}

func (n *Node) requestSync(target string, head string, events []hashgraph.Event) error {
	args := net.SyncRequest{
		Head:   head,
		Events: events,
	}
	var out net.SyncResponse
	if err := n.trans.Sync(target, &args, &out); err != nil {
		return err
	}

	n.logger.Printf("[DEBUG] node: %s replied to Sync with %#v", target, out.Success)
	return nil
}

// randomTimeout returns a value that is between the minVal and 2x minVal.
func randomTimeout(minVal time.Duration) <-chan time.Time {
	if minVal == 0 {
		return nil
	}
	extra := (time.Duration(rand.Int63()) % minVal)
	return time.After(minVal + extra)
}

func (n *Node) Shutdown() {
	n.shutdownLock.Lock()
	defer n.shutdownLock.Unlock()

	if !n.shutdown {
		n.logger.Printf("[DEBUG] node: %s shutdown", n.localAddr)
		close(n.shutdownCh)
		n.shutdown = true
	}
}

func (n *Node) GetConsensus() []string {
	consensus := n.core.GetConsensus()
	return consensus
}
