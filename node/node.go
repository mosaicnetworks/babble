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
	"sort"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"strconv"

	hg "bitbucket.org/mosaicnet/babble/hashgraph"
	"bitbucket.org/mosaicnet/babble/net"
	"bitbucket.org/mosaicnet/babble/proxy"
)

type Node struct {
	conf   *Config
	logger *logrus.Entry

	id       int
	core     *Core
	coreLock sync.Mutex

	localAddr string

	peerSelector PeerSelector
	selectorLock sync.Mutex

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

	busy      bool
	busyTimer *time.Timer
	busyLock  sync.Mutex
}

func NewNode(conf *Config, key *ecdsa.PrivateKey, participants []net.Peer, trans net.Transport, proxy proxy.Proxy) Node {
	localAddr := trans.LocalAddr()

	sort.Sort(net.ByPubKey(participants))
	pmap := make(map[string]int)
	var id int
	for i, p := range participants {
		pmap[p.PubKeyHex] = i
		if p.NetAddr == localAddr {
			id = i
		}
	}

	store := hg.NewInmemStore(pmap, conf.CacheSize)
	commitCh := make(chan []hg.Event, 20)
	core := NewCore(id, key, pmap, store, commitCh, conf.Logger)

	peerSelector := NewSmartPeerSelector(participants, localAddr)
	//peerSelector := NewRandomPeerSelector(participants, localAddr)

	node := Node{
		id:              id,
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
		busyTimer:       time.NewTimer(conf.TCPTimeout),
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
				go n.gossip()
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
		case <-n.busyTimer.C:
			n.SetBusy(false)
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

	var known map[int]int
	var err error

	if !n.IsBusy() {
		n.SetBusy(true)

		n.coreLock.Lock()
		known = n.core.Known()
		n.coreLock.Unlock()

		elapsed := time.Since(start)
		n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed Known()")
	} else {
		err = fmt.Errorf("Busy")
	}

	resp := &net.KnownResponse{
		Known: known,
	}
	rpc.Respond(resp, err)

}

func (n *Node) processSync(rpc net.RPC, cmd *net.SyncRequest) {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()
	defer n.SetBusy(false)

	success := true

	start := time.Now()
	err := n.core.Sync(cmd.Head, cmd.Events, n.transactionPool)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed Sync()")
	if err != nil {
		n.logger.WithField("error", err).Error("Sync()")
		success = false
	} else {
		n.transactionPool = [][]byte{}
		start = time.Now()
		err := n.core.RunConsensus()
		elapsed = time.Since(start)
		n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed RunConsensus()")
		if err != nil {
			n.logger.WithField("error", err).Error("RunConsensus()")
			success = false
		} else {
			n.UpdatePeerSelector(cmd.From)
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

		n.coreLock.Lock()
		head, diff, err := n.core.Diff(known)
		n.coreLock.Unlock()

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
		n.UpdatePeerSelector(peer.NetAddr)
	}
}

func (n *Node) requestKnown(target string) (map[int]int, error) {
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

	timeElapsed := time.Since(n.start)

	consensusEvents := n.core.GetConsensusEventsCount()
	consensusEventsPerSecond := float64(consensusEvents) / timeElapsed.Seconds()

	lastConsensusRound := n.core.GetLastConsensusRoundIndex()
	var consensusRoundsPerSecond float64
	if lastConsensusRound != nil {
		consensusRoundsPerSecond = float64(*lastConsensusRound) / timeElapsed.Seconds()
	}

	s := map[string]string{
		"last_consensus_round":   toString(lastConsensusRound),
		"consensus_events":       strconv.Itoa(consensusEvents),
		"consensus_transactions": strconv.Itoa(n.core.GetConsensusTransactionsCount()),
		"undetermined_events":    strconv.Itoa(len(n.core.GetUndeterminedEvents())),
		"transaction_pool":       strconv.Itoa(len(n.transactionPool)),
		"num_peers":              strconv.Itoa(len(n.peerSelector.Peers())),
		"sync_rate":              strconv.FormatFloat(n.SyncRate(), 'f', 2, 64),
		"events_per_second":      strconv.FormatFloat(consensusEventsPerSecond, 'f', 2, 64),
		"rounds_per_second":      strconv.FormatFloat(consensusRoundsPerSecond, 'f', 2, 64),
		"round_events":           strconv.Itoa(n.core.GetLastCommitedRoundEventsCount()),
		"id":                     strconv.Itoa(n.id),
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
		"rounds/s":               stats["rounds_per_second"],
		"round_events":           stats["round_events"],
		"id":                     stats["id"],
	}).Debug("Stats")
}

func (n *Node) SyncRate() float64 {
	var syncErrorRate float64
	if n.syncRequests != 0 {
		syncErrorRate = float64(n.syncErrors) / float64(n.syncRequests)
	}
	return 1 - syncErrorRate
}

func (n *Node) IsBusy() bool {
	n.busyLock.Lock()
	defer n.busyLock.Unlock()
	return n.busy
}

func (n *Node) SetBusy(val bool) {
	n.busyLock.Lock()
	defer n.busyLock.Unlock()
	n.busy = val
	if val {
		n.busyTimer.Reset(n.conf.TCPTimeout)
	}
}

func (n *Node) UpdatePeerSelector(peer string) {
	n.selectorLock.Lock()
	defer n.selectorLock.Unlock()
	n.peerSelector.Update(peer)
}

func randomTimeout(minVal time.Duration) <-chan time.Time {
	if minVal == 0 {
		return nil
	}
	extra := (time.Duration(rand.Int63()) % minVal)
	return time.After(minVal + extra)
}
