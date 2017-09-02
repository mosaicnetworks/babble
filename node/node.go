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

	proxy    proxy.AppProxy
	submitCh chan []byte

	commitCh chan []hg.Event

	// Shutdown channel to exit, protected to prevent concurrent exits
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	start        time.Time
	syncRequests int
	syncErrors   int
}

func NewNode(conf *Config, key *ecdsa.PrivateKey, participants []net.Peer, trans net.Transport, proxy proxy.AppProxy) Node {
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
		id:           id,
		conf:         conf,
		core:         &core,
		localAddr:    localAddr,
		logger:       conf.Logger.WithField("node", localAddr),
		peerSelector: peerSelector,
		trans:        trans,
		netCh:        trans.Consumer(),
		proxy:        proxy,
		submitCh:     proxy.SubmitCh(),
		commitCh:     commitCh,
		shutdownCh:   make(chan struct{}),
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
			if n.core.NeedGossip() && heartbeatTimer == nil {
				heartbeatTimer = randomTimeout(n.conf.HeartbeatTimeout)
			}
		case <-heartbeatTimer:
			if gossip {
				proceed, err := n.preGossip()
				if proceed && err == nil {
					n.logger.Debug("Time to gossip!")
					peer := n.peerSelector.Next()
					go n.gossip(peer.NetAddr)
				}
			}
			if n.core.NeedGossip() {
				heartbeatTimer = randomTimeout(n.conf.HeartbeatTimeout)
			} else {
				heartbeatTimer = nil
			}
		case t := <-n.submitCh:
			n.logger.Debug("Adding Transaction")
			n.addTransaction(t)
			if heartbeatTimer == nil {
				heartbeatTimer = randomTimeout(n.conf.HeartbeatTimeout)
			}
		case events := <-n.commitCh:
			n.logger.WithField("events", len(events)).Debug("Committing Events")
			if err := n.commit(events); err != nil {
				n.logger.WithField("error", err).Error("Committing Event")
			}
		case <-n.shutdownCh:
			return
		}
	}
}

func (n *Node) processRPC(rpc net.RPC) {
	switch cmd := rpc.Command.(type) {
	case *net.SyncRequest:
		n.processSyncRequest(rpc, cmd)
	case *net.EagerSyncRequest:
		n.processEagerSyncRequest(rpc, cmd)
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

	//Compute Diff
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

	//Convert to WireEvents
	wireEvents, err := n.core.ToWire(diff)
	if err != nil {
		n.logger.WithField("error", err).Debug("Converting to WireEvent")
		return
	}

	//Get Self Known
	n.coreLock.Lock()
	known := n.core.Known()
	n.coreLock.Unlock()

	//Build SyncResponse and respond
	n.logger.WithField("events", len(diff)).Debug("Responding to SyncRequest")
	resp := &net.SyncResponse{
		From:   n.localAddr,
		Head:   head,
		Events: wireEvents,
		Known:  known,
	}
	rpc.Respond(resp, err)
}

func (n *Node) processEagerSyncRequest(rpc net.RPC, cmd *net.EagerSyncRequest) {
	n.logger.WithFields(logrus.Fields{
		"from":   cmd.From,
		"events": len(cmd.Events),
	}).Debug("EagerSyncRequest")

	success := true

	err := n.sync(cmd.Head, cmd.Events)
	if err != nil {
		n.logger.WithField("error", err).Error("sync()")
		success = false
	}

	resp := &net.EagerSyncResponse{
		Success: success,
	}
	rpc.Respond(resp, err)
}

func (n *Node) preGossip() (bool, error) {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	//Check if it is necessary to gossip
	needGossip := n.core.NeedGossip()
	if !needGossip {
		n.logger.Debug("Nothing to gossip")
		return false, nil
	}

	//If the transaction pool is not empty, create a new self-event and empty the
	//transaction pool in its payload
	if err := n.core.AddSelfEvent(); err != nil {
		n.logger.WithField("error", err).Error("Adding SelfEvent")
		return false, err
	}

	return true, nil
}

func (n *Node) gossip(peerAddr string) error {
	//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	//PULL
	//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

	//Compute Known
	n.coreLock.Lock()
	known := n.core.Known()
	n.coreLock.Unlock()

	//Send SyncRequest
	start := time.Now()
	resp, err := n.requestSync(peerAddr, known)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestSync()")
	if err != nil {
		n.logger.WithField("error", err).Error("requestSync()")
		return err
	}
	n.logger.WithFields(logrus.Fields{
		"events": fmt.Sprintf("%#v", resp.Events),
		"known":  fmt.Sprintf("%#v", resp.Known),
	}).Debug("SyncResponse")

	//Add Events to Hashgraph and create new Head if necessary
	err = n.sync(resp.Head, resp.Events)
	if err != nil {
		n.logger.WithField("error", err).Error("sync()")
		return err
	}

	//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	//PUSH
	//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

	//Compute Diff
	start = time.Now()
	n.coreLock.Lock()
	head, diff, err := n.core.Diff(resp.Known)
	n.coreLock.Unlock()
	elapsed = time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Diff()")
	if err != nil {
		n.logger.WithField("error", err).Error("Calculating Diff")
		return err
	}

	//Convert to WireEvents
	wireEvents, err := n.core.ToWire(diff)
	if err != nil {
		n.logger.WithField("error", err).Debug("Converting to WireEvent")
		return err
	}

	//Create and Send EagerSyncRequest
	start = time.Now()
	resp2, err := n.requestEagerSync(peerAddr, head, wireEvents)
	elapsed = time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("requestEagerSync()")
	if err != nil {
		n.logger.WithField("error", err).Error("requestEagerSync()")
		return err
	}
	n.logger.WithField("success", resp2.Success).Debug("EagerSyncResponse")

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

func (n *Node) requestEagerSync(target string, head string, events []hg.WireEvent) (net.EagerSyncResponse, error) {
	args := net.EagerSyncRequest{
		From:   n.localAddr,
		Head:   head,
		Events: events,
	}

	var out net.EagerSyncResponse
	err := n.trans.EagerSync(target, &args, &out)

	return out, err
}

func (n *Node) sync(head string, events []hg.WireEvent) error {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()

	//Insert Events in Hashgraph and create new Head if necessary
	start := time.Now()
	err := n.core.Sync(head, events)
	elapsed := time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed Sync()")
	if err != nil {
		return err
	}

	//Run consensus methods
	start = time.Now()
	err = n.core.RunConsensus()
	elapsed = time.Since(start)
	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Processed RunConsensus()")
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) commit(events []hg.Event) error {
	for _, ev := range events {
		for _, tx := range ev.Transactions() {
			if err := n.proxy.CommitTx(tx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Node) addTransaction(tx []byte) {
	n.coreLock.Lock()
	defer n.coreLock.Unlock()
	n.core.AddTransactions([][]byte{tx})
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
		"transaction_pool":       strconv.Itoa(len(n.core.transactionPool)),
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
