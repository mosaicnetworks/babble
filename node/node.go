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

	hg "bitbucket.org/mosaicnet/babble/hashgraph"
	"bitbucket.org/mosaicnet/babble/net"
	"bitbucket.org/mosaicnet/babble/proxy"
)

type Node struct {
	conf *Config

	core *Core

	localAddr string
	logger    *logrus.Entry

	peerSelector PeerSelector

	trans net.Transport
	netCh <-chan net.RPC

	proxy    proxy.Proxy
	submitCh chan []byte

	commitCh chan []hg.Event

	// Shutdown channel to exit, protected to prevent concurrent exits
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	transactionPool [][]byte

	start        time.Time
	syncRequests int
	syncErrors   int
}

func NewNode(conf *Config, key *ecdsa.PrivateKey, participants []net.Peer, trans net.Transport, proxy proxy.Proxy) Node {
	localAddr := trans.LocalAddr()

	participantPubs := []string{}
	for _, p := range participants {
		participantPubs = append(participantPubs, p.PubKeyHex)
	}
	store := hg.NewInmemStore(participantPubs)
	commitCh := make(chan []hg.Event, 20)
	core := NewCore(key, participantPubs, store, commitCh, conf.Logger)

	peerSelector := NewDeterministicPeerSelector(participants, localAddr)

	node := Node{
		conf:            conf,
		core:            &core,
		localAddr:       localAddr,
		logger:          conf.Logger.WithField("node", localAddr),
		peerSelector:    peerSelector,
		trans:           trans,
		netCh:           trans.Consumer(),
		proxy:           proxy,
		submitCh:        proxy.Consumer(),
		commitCh:        commitCh,
		shutdownCh:      make(chan struct{}),
		transactionPool: [][]byte{},
	}
	return node
}

func (n *Node) Init() error {
	peerAddresses := []string{}
	for _, p := range n.peerSelector.Peers() {
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
	n.start = time.Now()
	heartbeatTimer := randomTimeout(n.conf.HeartbeatTimeout)
	for {
		select {
		case rpc := <-n.netCh:
			n.logger.Debug("Processing RPC")
			n.processRPC(rpc)
		case <-heartbeatTimer:
			if gossip {
				n.logger.Debug("Time to gossip!")
				n.gossip()
			}
			heartbeatTimer = randomTimeout(n.conf.HeartbeatTimeout)
		case t := <-n.submitCh:
			n.logger.Debug("Adding Transaction")
			n.transactionPool = append(n.transactionPool, t)
		case events := <-n.commitCh:
			n.logger.WithField("events", len(events)).Debug("Committing Events")
			if err := n.Commit(events); err != nil {
				n.logger.WithField("error", err).Error("Committing Event")
			}
		case <-n.shutdownCh:
			return
		default:
		}
	}
}

func (n *Node) processRPC(rpc net.RPC) {
	switch cmd := rpc.Command.(type) {
	case *net.KnownRequest:
		n.logger.WithField("from", cmd.From).Debug("Processing Known")
		n.processKnown(rpc, cmd)
	case *net.SyncRequest:
		n.logger.WithField("from", cmd.From).Debug("Processing Sync")
		n.processSync(rpc, cmd)
	default:
		n.logger.WithField("cmd", rpc.Command).Error("Unexpected RPC command")
		rpc.Respond(nil, fmt.Errorf("unexpected command"))
	}
}

func (n *Node) processKnown(rpc net.RPC, cmd *net.KnownRequest) {
	start := time.Now()
	known := n.core.Known()
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed).Debug("Processed Known()")
	resp := &net.KnownResponse{
		Known: known,
	}
	rpc.Respond(resp, nil)
}

func (n *Node) processSync(rpc net.RPC, cmd *net.SyncRequest) {
	success := true
	start := time.Now()
	err := n.core.Sync(cmd.Head, cmd.Events, n.transactionPool)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed).Debug("Processed Sync()")
	if err != nil {
		n.logger.WithField("error", err).Error("Sync()")
		success = false
	} else {
		n.transactionPool = [][]byte{}
		start = time.Now()
		err := n.core.RunConsensus()
		elapsed = time.Since(start)
		n.logger.WithField("duration", elapsed).Debug("Processed RunConsensus()")
		if err != nil {
			n.logger.WithField("error", err).Error("RunConsensus()")
			success = false
		}
	}
	resp := &net.SyncResponse{
		Success: success,
	}
	rpc.Respond(resp, err)
	n.logStats()
}

func (n *Node) gossip() {
	peer := n.peerSelector.Next()

	known, err := n.requestKnown(peer.NetAddr)
	if err != nil {
		n.logger.WithField("error", err).Error("Getting peer Known")
	} else {
		head, diff, err := n.core.Diff(known)
		if err != nil {
			n.logger.WithField("error", err).Error("Calculating Diff")
			return
		}
		n.syncRequests++
		err = n.requestSync(peer.NetAddr, head, diff)
		if err != nil {
			n.syncErrors++
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

func (n *Node) requestSync(target string, head string, events []hg.Event) error {
	args := net.SyncRequest{
		From:   n.localAddr,
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

func (n *Node) Commit(events []hg.Event) error {
	for _, ev := range events {
		for _, tx := range ev.Transactions() {
			if err := n.proxy.CommitTx(tx); err != nil {
				return err
			}
		}
	}
	return nil
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

func (n *Node) GetStats() map[string]string {
	toString := func(i *int) string {
		if i == nil {
			return "nil"
		}
		return strconv.Itoa(*i)
	}

	consensusEvents := n.core.GetConsensusEventsCount()
	timeElapsed := time.Since(n.start)
	consensusEventsPerSecond := float64(consensusEvents) / timeElapsed.Seconds()

	s := map[string]string{
		"last_consensus_round":   toString(n.core.GetLastConsensusRoundIndex()),
		"consensus_events":       strconv.Itoa(consensusEvents),
		"consensus_transactions": strconv.Itoa(n.core.GetConsensusTransactionsCount()),
		"undetermined_events":    strconv.Itoa(len(n.core.GetUndeterminedEvents())),
		"transaction_pool":       strconv.Itoa(len(n.transactionPool)),
		"num_peers":              strconv.Itoa(len(n.peerSelector.Peers())),
		"sync_rate":              strconv.FormatFloat(n.SyncRate(), 'f', 2, 64),
		"events_per_second":      strconv.FormatFloat(consensusEventsPerSecond, 'f', 2, 64),
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
		"sync_rate":              stats["sync_rate"],
		"events/s":               stats["events_per_second"],
	}).Debug("Stats")
}

func (n *Node) SyncRate() float64 {
	var syncErrorRate float64
	if n.syncRequests != 0 {
		syncErrorRate = float64(n.syncErrors) / float64(n.syncRequests)
	}
	return 1 - syncErrorRate
}

func randomTimeout(minVal time.Duration) <-chan time.Time {
	if minVal == 0 {
		return nil
	}
	extra := (time.Duration(rand.Int63()) % minVal)
	return time.After(minVal + extra)
}
