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

	hg "github.com/babbleio/babble/hashgraph"
	"github.com/babbleio/babble/net"
	"github.com/babbleio/babble/proxy"
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

	peerSelector := NewRandomPeerSelector(participants, localAddr)

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
				peer := n.peerSelector.Next()
				go n.gossip(peer.NetAddr)
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
	case *net.SyncRequest:
		n.logger.WithField("from", cmd.From).Debug("Processing SyncRequest")
		n.processSyncRequest(rpc, cmd)
	default:
		n.logger.WithField("cmd", rpc.Command).Error("Unexpected RPC command")
		rpc.Respond(nil, fmt.Errorf("unexpected command"))
	}
}

func (n *Node) processSyncRequest(rpc net.RPC, cmd *net.SyncRequest) {
	n.logger.WithFields(logrus.Fields{
		"from":  cmd.From,
		"known": cmd.Known,
	}).Debug("SyncRequest")

	start := time.Now()
	n.coreLock.Lock()
	head, diff, err := n.core.Diff(cmd.Known)
	n.coreLock.Unlock()
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Diff()")

	if err != nil {
		n.logger.WithField("error", err).Error("Calculating Diff")
		return
	}

	wireEvents, err := n.core.ToWire(diff)
	if err != nil {
		n.logger.WithField("error", err).Debug("Converting to WireEvent")
		return
	}

	n.logger.WithField("events", len(diff)).Debug("Responding to Sync Request")
	resp := &net.SyncResponse{
		From:   n.localAddr,
		Head:   head,
		Events: wireEvents,
	}
	rpc.Respond(resp, err)
}

func (n *Node) gossip(peerAddr string) error {

	n.coreLock.Lock()
	known := n.core.Known()
	n.coreLock.Unlock()

	start := time.Now()
	resp, err := n.requestSync(peerAddr, known)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestSync()")
	if err != nil {
		n.logger.WithField("error", err).Error("requestSync()")
		return err
	}

	err = n.processSyncResponse(resp)
	if err != nil {
		n.logger.WithField("error", err).Error("processSyncResponse()")
		return err
	}

	n.selectorLock.Lock()
	n.peerSelector.UpdateLast(peerAddr)
	n.selectorLock.Unlock()

	n.logStats()

	return nil

}

func (n *Node) requestSync(target string, known map[int]int) (net.SyncResponse, error) {
	args := net.SyncRequest{
		From:  n.localAddr,
		Known: known,
	}

	var out net.SyncResponse
	err := n.trans.Sync(target, &args, &out)

	return out, err
}

func (n *Node) processSyncResponse(resp net.SyncResponse) error {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	n.logger.WithField("events", fmt.Sprintf("%#v", resp.Events)).Debug("SyncResponse")

	start := time.Now()

	err := n.core.Sync(resp.Head, resp.Events, n.transactionPool)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed Sync()")
	if err != nil {
		return err
	}

	n.transactionPool = [][]byte{}
	start = time.Now()
	err = n.core.RunConsensus()
	elapsed = time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed RunConsensus()")
	if err != nil {
		return err
	}

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

func randomTimeout(minVal time.Duration) <-chan time.Time {
	if minVal == 0 {
		return nil
	}
	extra := (time.Duration(rand.Int63()) % minVal)
	return time.After(minVal + extra)
}
