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
	"math/rand"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"strconv"

	"github.com/arrivets/babble/hashgraph"
	"github.com/arrivets/babble/net"
)

type Node struct {
	conf *Config

	core *Core

	localAddr string
	logger    *logrus.Entry

	peers []net.Peer

	// RPC chan comes from the transport layer
	rpcCh <-chan net.RPC

	txCh chan []byte

	// Shutdown channel to exit, protected to prevent concurrent exits
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	// The transport layer we use
	trans net.Transport

	transactionPool [][]byte
}

func NewNode(conf *Config, key *ecdsa.PrivateKey, participants []net.Peer, trans net.Transport) Node {
	localAddr := trans.LocalAddr()

	participantPubs := []string{}
	for _, p := range participants {
		participantPubs = append(participantPubs, p.PubKeyHex)
	}
	core := NewCore(key, participantPubs)

	peers := net.ExcludePeer(participants, localAddr)

	node := Node{
		conf:            conf,
		core:            &core,
		localAddr:       localAddr,
		logger:          conf.Logger.WithField("node", localAddr),
		peers:           peers,
		rpcCh:           trans.Consumer(),
		txCh:            make(chan []byte),
		shutdownCh:      make(chan struct{}),
		trans:           trans,
		transactionPool: [][]byte{},
	}
	return node
}

func (n *Node) Init() error {
	peerAddresses := []string{}
	for _, p := range n.peers {
		peerAddresses = append(peerAddresses, p.NetAddr)
	}
	n.logger.WithField("peers", peerAddresses).Debug("Init Node")
	return n.core.Init()
}

func (n *Node) RunAsync(gossip bool) {
	n.logger.Debug("runasync")
	go n.Run(gossip)
}

func (n *Node) Run(gossip bool) {
	heartbeatTimer := randomTimeout(n.conf.HeartbeatTimeout)
	for {
		select {
		case rpc := <-n.rpcCh:
			n.logger.Debug("Processing RPC")
			n.processRPC(rpc)
		case <-heartbeatTimer:
			if gossip {
				n.logger.Debug("Time to gossip!")
				n.gossip()
			}
			heartbeatTimer = randomTimeout(n.conf.HeartbeatTimeout)
		case t := <-n.txCh:
			n.logger.Debug("Adding Transaction")
			n.transactionPool = append(n.transactionPool, t)
		case <-n.shutdownCh:
			return
		default:
		}
	}
}

func (n *Node) processRPC(rpc net.RPC) {
	switch cmd := rpc.Command.(type) {
	case *net.KnownRequest:
		n.logger.Debug("processing known")
		n.processKnown(rpc, cmd)
	case *net.SyncRequest:
		n.logger.Debug("processing sync")
		n.processSync(rpc, cmd)
	default:
		n.logger.WithField("cmd", rpc.Command).Error("Unexpected RPC command")
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
	err := n.core.Sync(cmd.Head, cmd.Events, n.transactionPool)
	if err != nil {
		success = false
	} else {
		n.transactionPool = [][]byte{}
		n.core.RunConsensus()
	}
	resp := &net.SyncResponse{
		Success: success,
	}
	rpc.Respond(resp, err)
	n.logStats()
}

func (n *Node) gossip() {
	i := rand.Intn(len(n.peers))
	peer := n.peers[i]

	known, err := n.requestKnown(peer.NetAddr)
	if err != nil {
		n.logger.WithField("error", err).Error("Getting peer Known")
	} else {
		head, diff := n.core.Diff(known)
		if err := n.requestSync(peer.NetAddr, head, diff); err != nil {
			n.logger.WithField("error", err).Error("Triggering peer Sync")
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
	n.logger.WithFields(logrus.Fields{
		"target": target,
		"known":  out.Known,
	}).Debug("Known Response")
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
	n.logger.WithFields(logrus.Fields{
		"target":  target,
		"success": out.Success,
	}).Debug("Sync Response")
	return nil
}

func (n *Node) AddTransaction(t []byte) {
	n.txCh <- t
}

func (n *Node) Shutdown() {
	n.shutdownLock.Lock()
	defer n.shutdownLock.Unlock()

	if !n.shutdown {
		n.logger.Debug("Shutdown")
		close(n.shutdownCh)
		n.shutdown = true
	}
}

func (n *Node) GetConsensusEvents() []string {
	return n.core.GetConsensusEvents()
}

func (n *Node) GetConsensusTransactions() ([][]byte, error) {
	return n.core.GetConsensusTransactions()
}

func (n *Node) GetStats() map[string]string {
	toString := func(i *int) string {
		if i == nil {
			return "nil"
		}
		return strconv.Itoa(*i)
	}
	s := map[string]string{
		"last_consensus_round":   toString(n.core.GetLastConsensusRoundIndex()),
		"consensus_events":       strconv.Itoa(len(n.core.GetConsensusEvents())),
		"consensus_transactions": strconv.Itoa(n.core.GetConsensusTransactionsCount()),
		"undetermined_events":    strconv.Itoa(len(n.core.GetUndeterminedEvents())),
		"transaction_pool":       strconv.Itoa(len(n.transactionPool)),
		"num_peers":              strconv.Itoa(len(n.peers)),
	}
	return s
}

func (n *Node) logStats() {
	stats := n.GetStats()
	n.logger.WithFields(logrus.Fields{
		"last_consensus_round":   stats["last_consensus_round"],
		"consensus_events":       stats["consensus_events"],
		"consensus_transactions": stats["consensus_transactions"],
		"undetermined_events":    stats["undetermined_events"],
		"transaction_pool":       stats["transaction_pool"],
		"num_peers":              stats["num_peers"],
	}).Debug("Stats")
}

func randomTimeout(minVal time.Duration) <-chan time.Time {
	if minVal == 0 {
		return nil
	}
	extra := (time.Duration(rand.Int63()) % minVal)
	return time.After(minVal + extra)
}
