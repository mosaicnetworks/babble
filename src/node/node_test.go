package node

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/config"
	bkeys "github.com/mosaicnetworks/babble/src/crypto/keys"
	dummy "github.com/mosaicnetworks/babble/src/dummy"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*

Tests for regular gossip routines when fast-sync is disabled.

NO FAST-SYNC, NO DYNAMIC PARTICIPANTS.

*/
var ip = 9990

//config for test WebRTC signaling service
var certFile = "../net/signal/wamp/test_data/cert.pem"
var keyFile = "../net/signal/wamp/test_data/key.pem"
var realm = config.DefaultSignalRealm

func TestAddTransaction(t *testing.T) {
	keys, peers := initPeers(t, 2)
	genesisPeerSet := clonePeerSet(t, peers.Peers)

	//Start two nodes
	nodes := initNodes(
		keys,
		peers,
		genesisPeerSet,
		10000,
		500,
		1*time.Second,
		false,
		"inmem",
		10*time.Millisecond,
		false,
		"",
		t)
	defer shutdownNodes(nodes)

	nodes[0].RunAsync(false)
	nodes[1].RunAsync(false)

	//Submit a Tx to node0
	message := "Hello World!"
	node0AppProxy := nodes[0].proxy.(*dummy.InmemDummyClient)
	node0AppProxy.SubmitTx([]byte(message))

	//simulate a SyncRequest from node0 to node1
	node0KnownEvents := nodes[0].core.knownEvents()

	resp, err := nodes[0].requestSync(
		peers.Peers[1].NetAddr,
		node0KnownEvents,
		500)
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}

	if err := nodes[0].sync(peers.Peers[1].ID(), resp.Events); err != nil {
		t.Error("Fatal Error 3", err)
		t.Fatal(err)
	}

	//check the Tx was removed from the transactionPool and added to the new Head

	if l := len(nodes[0].core.transactionPool); l > 0 {
		t.Fatalf("Fatal node0's transactionPool should have 0 elements, not %d\n", l)
	}

	node0Head, _ := nodes[0].core.getHead()
	if l := len(node0Head.Transactions()); l != 1 {
		t.Fatalf("Fatal node0's Head should have 1 element, not %d\n", l)
	}

	if m := string(node0Head.Transactions()[0]); m != message {
		t.Fatalf("Fatal Transaction message should be '%s' not, not %s\n", message, m)
	}
}

func TestGossip(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 100000, 1000, 5, false, "inmem", 5*time.Millisecond, false, "", t)
	//defer drawGraphs(nodes, t)

	target := 50
	err := gossip(nodes, target, true)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}

	checkGossip(nodes, 0, t)

	checkTimestamps(nodes[0], t)
}

func TestWebRTCGossip(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	server, err := wamp.NewServer(
		"localhost:2443",
		realm,
		certFile,
		keyFile,
		common.NewTestEntry(t, common.TestLogLevel),
	)
	if err != nil {
		t.Fatal(err)
	}

	go server.Run()
	defer server.Shutdown()
	time.Sleep(time.Second)

	nodes := initNodes(
		keys,
		peers,
		genesisPeerSet,
		100000,
		1000,
		10,
		false,
		"inmem",
		50*time.Millisecond, // slow down for webrtc
		true,
		"localhost:2443",
		t,
	)
	//defer drawGraphs(nodes, t)

	target := 50
	err = gossip(nodes, target, true)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}

	checkGossip(nodes, 0, t)
}

func TestMissingNodeGossip(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, false, "inmem", 5*time.Millisecond, false, "", t)
	//defer drawGraphs(nodes, t)

	err := gossip(nodes[1:], 10, true)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}

	checkGossip(nodes[1:], 0, t)
}

func TestSyncLimit(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, false, "inmem", 5*time.Millisecond, false, "", t)
	defer shutdownNodes(nodes)

	err := gossip(nodes, 10, false)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}

	//create fake node[0] known to artificially reach SyncLimit
	node0KnownEvents := nodes[0].core.knownEvents()
	for k := range node0KnownEvents {
		node0KnownEvents[k] = 0
	}

	//create a sync request with a low SyncLimit of 50. The responding node
	//should account for the SyncLimit and return only 50 events.
	args := net.SyncRequest{
		FromID:    nodes[0].core.validator.ID(),
		SyncLimit: 50,
		Known:     node0KnownEvents,
	}

	var out net.SyncResponse
	if err := nodes[0].trans.Sync(nodes[1].trans.LocalAddr(), &args, &out); err != nil {
		t.Fatalf("Fatal err: %v", err)
	}

	//Check that response contains only 50 events
	if len(out.Events) != 50 {
		t.Fatalf("Fatal SyncResponse should contain 50 events, not %d", len(out.Events))
	}
}

func TestShutdown(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, false, "inmem", 5*time.Millisecond, false, "", t)
	runNodes(nodes, false)
	defer shutdownNodes(nodes)

	nodes[0].Shutdown()
	err := nodes[1].gossip(peers.Peers[0])
	if err == nil {
		t.Fatal("Fatal Expected Timeout Error")
	}
}

func TestBootstrapAllNodes(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	// create a first network with BadgerStore and wait till it reaches 10
	// blocks before shutting it down
	keys, peers := initPeers(t, 4)
	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 100000, 1000, 10, false, "badger", 10*time.Millisecond, false, "", t)

	err := gossip(nodes, 10, true)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	// Now try to recreate a network from the databases created in the first
	// step and advance it to 20 blocks
	newNodes := recycleNodes(nodes, t)

	err = gossip(newNodes, 20, true)
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}
	checkGossip(newNodes, 0, t)

	//Check that both networks did not have completely different consensus events
	checkGossip([]*Node{nodes[0], newNodes[0]}, 0, t)
}

func BenchmarkGossip(b *testing.B) {
	for n := 0; n < b.N; n++ {
		keys, peers := initPeers(b, 4)

		genesisPeerSet := clonePeerSet(b, peers.Peers)

		nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, false, "inmem", 5*time.Millisecond, false, "", b)

		gossip(nodes, 50, true)
	}
}

/*******************************************************************************
HELPERS
*******************************************************************************/

func initPeers(t testing.TB, n int) ([]*ecdsa.PrivateKey, *peers.PeerSet) {
	keys := []*ecdsa.PrivateKey{}
	pirs := []*peers.Peer{}

	for i := 0; i < n; i++ {
		key, _ := bkeys.GenerateECDSAKey()
		keys = append(keys, key)
		peer := peers.NewPeer(
			bkeys.PublicKeyHex(&keys[i].PublicKey),
			fmt.Sprintf("127.0.0.1:%d", ip),
			fmt.Sprintf("node%d", i),
		)
		pirs = append(pirs, peer)
		if t != nil {
			t.Logf("Setting up Node %d on 127.0.0.1:%d  %s", i, ip, bkeys.PublicKeyHex(&keys[i].PublicKey))
		}
		ip++
	}

	peerSet := peers.NewPeerSet(pirs)

	return keys, peerSet
}

func clonePeerSet(t testing.TB, sourcePeers []*peers.Peer) *peers.PeerSet {
	var newPeers []*peers.Peer
	for _, p := range sourcePeers {
		newPeers = append(newPeers, peers.NewPeer(p.PubKeyHex, p.NetAddr, p.Moniker))
	}

	return peers.NewPeerSet(newPeers)
}

func newNode(peer *peers.Peer,
	k *ecdsa.PrivateKey,
	peers *peers.PeerSet,
	genesisPeers *peers.PeerSet,
	cacheSize,
	syncLimit int,
	joinTimeoutSeconds time.Duration,
	enableSyncLimit bool,
	storeType string,
	heartbeatTimeout time.Duration,
	webrtc bool,
	signalServer string,
	t testing.TB) *Node {

	conf := config.NewTestConfig(t, common.TestLogLevel)
	conf.HeartbeatTimeout = heartbeatTimeout
	conf.MaxPool = 3
	conf.TCPTimeout = 2 * time.Second
	conf.JoinTimeout = joinTimeoutSeconds * time.Second
	conf.CacheSize = cacheSize
	conf.SyncLimit = syncLimit
	conf.EnableFastSync = enableSyncLimit

	t.Logf("Starting node on %s", peer.NetAddr)

	var trans net.Transport
	var err error

	if !webrtc {
		trans, err = net.NewTCPTransport(
			peer.NetAddr,
			"",
			conf.MaxPool,
			conf.TCPTimeout,
			conf.JoinTimeout,
			conf.Logger(),
		)
		if err != nil {
			t.Fatalf("Fatal failed to create transport for peer %d: %s", peer.ID(), err)
		}
	} else {
		signal, err := wamp.NewClient(
			signalServer,
			realm,
			peer.NetAddr,
			certFile,
			false,
			conf.TCPTimeout,
			common.NewTestEntry(t, common.TestLogLevel),
		)

		if err != nil {
			t.Fatal(err)
		}

		trans, err = net.NewWebRTCTransport(
			signal,
			conf.ICEServers(),
			conf.MaxPool,
			conf.TCPTimeout,
			conf.JoinTimeout,
			conf.Logger(),
		)

		if err != nil {
			t.Fatal(err)
		}
	}

	go trans.Listen()

	var store hg.Store
	switch storeType {
	case "badger":
		path, _ := ioutil.TempDir("test_data", "badger")
		store, err = hg.NewBadgerStore(conf.CacheSize, path, false, nil)
		if err != nil {
			t.Fatalf("Fatal failed to create BadgerStore for peer %d: %s", peer.ID(), err)
		}
	case "inmem":
		store = hg.NewInmemStore(conf.CacheSize)
	}

	prox := dummy.NewInmemDummyClient(common.NewTestEntry(t, common.TestLogLevel))
	node := NewNode(conf,
		NewValidator(k, peer.Moniker),
		peers,
		genesisPeers,
		store,
		trans,
		prox)

	if err := node.Init(); err != nil {
		t.Fatalf("Fatal failed to initialize node%d: %s", peer.ID(), err)
	}

	t.Logf("Created Node %s %d", peer.Moniker, peer.ID())
	logPeerList(t, peers, "Peerlist: ")
	return node
}

func initNodes(keys []*ecdsa.PrivateKey,
	peers *peers.PeerSet,
	genesisPeers *peers.PeerSet,
	cacheSize,
	syncLimit int,
	joinTimeoutSeconds time.Duration,
	enableSyncLimit bool,
	storeType string,
	heartbeatTimeout time.Duration,
	webrtc bool,
	signalAddress string,
	t testing.TB) []*Node {

	nodes := []*Node{}

	for _, k := range keys {
		pubKey := bkeys.PublicKeyHex(&k.PublicKey)

		peer, ok := peers.ByPubKey[pubKey]
		if !ok {
			t.Fatalf("Fatal Peer not found")
		}

		node := newNode(peer,
			k,
			peers,
			genesisPeers,
			cacheSize,
			syncLimit,
			joinTimeoutSeconds,
			enableSyncLimit,
			storeType,
			heartbeatTimeout,
			webrtc,
			signalAddress,
			t)

		nodes = append(nodes, node)
	}
	return nodes
}

func recycleNodes(oldNodes []*Node, t *testing.T) []*Node {
	newNodes := []*Node{}
	for _, oldNode := range oldNodes {
		newNode := recycleNode(oldNode, t)
		newNodes = append(newNodes, newNode)
	}
	return newNodes
}

func recycleNode(oldNode *Node, t *testing.T) *Node {
	conf := oldNode.conf
	key := oldNode.core.validator.Key
	moniker := oldNode.core.validator.Moniker
	peers := oldNode.core.peers
	genesisPeerSet := oldNode.core.genesisPeers

	var store hg.Store
	var err error
	if _, ok := oldNode.core.hg.Store.(*hg.BadgerStore); ok {
		store, err = hg.NewBadgerStore(conf.CacheSize, oldNode.core.hg.Store.StorePath(), false, nil)
		if err != nil {
			t.Error("Fatal Error recyleNode", err)
			t.Fatal(err)
		}
	} else {
		store = hg.NewInmemStore(conf.CacheSize)
	}

	trans, err := net.NewTCPTransport(oldNode.trans.LocalAddr(),
		"", 2, conf.TCPTimeout, conf.JoinTimeout, conf.Logger())
	if err != nil {
		t.Error("Fatal Error 2 recycleNode", err)
		t.Fatal(err)
	}

	go trans.Listen()
	prox := dummy.NewInmemDummyClient(common.NewTestEntry(t, common.TestLogLevel))

	conf.Bootstrap = true

	newNode := NewNode(conf, NewValidator(key, moniker), peers, genesisPeerSet,
		store, trans, prox)

	if err := newNode.Init(); err != nil {
		t.Error("Fatal Error 3 recycleNode", err)
		t.Fatal(err)
	}

	return newNode
}

func runNodes(nodes []*Node, gossip bool) {
	for _, n := range nodes {
		node := n
		go func() {
			node.Run(gossip)
		}()
	}
}

func gossip(nodes []*Node, target int, shutdown bool) error {
	runNodes(nodes, true)
	err := bombardAndWait(nodes, target)
	if err != nil {
		return err
	}
	if shutdown {
		shutdownNodes(nodes)
	}
	return nil
}

func bombardAndWait(nodes []*Node, target int) error {
	//send a lot of random transactions to the nodes
	quit := make(chan struct{})
	makeRandomTransactions(nodes, quit)

	//wait until all nodes reach at least block 'target'

	// This function has been refactored. The original version specified a
	// timeout for reaching block number target. This meant that if a node
	// stalled, you would need to wait for the entire timeout period before
	// the error is flagged. Additionally, the timeout needed to be geared for
	// the slowest configuration of machine likely to run the test - which
	// lead to inflation of the timeout times.
	//
	// The revised version instead specifies a timeout for an individual node
	// to increment its block number. The outer loops ticks every 3 seconds
	// and if the block numbers for each node (nodeBlockNums) do not
	// progress then it is a failure.

	nodelength := len(nodes)
	nodeBlockNums := make([]int, nodelength)
	for i, n := range nodes {
		nodeBlockNums[i] = n.core.getLastBlockIndex()
	}

OUTERFOR:
	for {

		stopper := time.After(3 * time.Second)

	INNERFOR:
		for {
			select {
			case <-stopper:

				for i, n := range nodes {
					ce := n.core.getLastBlockIndex()
					if ce <= nodeBlockNums[i] {
						return fmt.Errorf("TIMEOUT in bombardAndWait node %d waiting for block %d, stalled at %d",
							i, target, ce)
					}
					nodeBlockNums[i] = ce
				}

				break INNERFOR
			default:
			}
			time.Sleep(10 * time.Millisecond)
			done := true
			for _, n := range nodes {
				ce := n.core.getLastBlockIndex()
				if ce < target {
					done = false
					break
				} else {
					//wait until the target block has retrieved a state hash from
					//the app
					targetBlock, err := n.core.hg.Store.GetBlock(target)
					if err != nil {
						return fmt.Errorf("Error: Couldn't find target block: %v, ce: %d", err, ce)
					}
					if len(targetBlock.StateHash()) == 0 {
						done = false
						break
					}
				}
			}

			if done {
				break OUTERFOR
			}
		}
	}

	close(quit)
	return nil
}

func makeRandomTransactions(nodes []*Node, quit chan struct{}) {
	go func() {
		seq := make(map[int]int)
		for {
			select {
			case <-quit:
				return
			default:
				n := rand.Intn(len(nodes))
				node := nodes[n]
				submitTransaction(node, []byte(fmt.Sprintf("node%d transaction %d", n, seq[n])))
				seq[n] = seq[n] + 1
				time.Sleep(3 * time.Millisecond)
			}
		}
	}()
}

func submitTransaction(n *Node, tx []byte) error {
	prox, ok := n.proxy.(*dummy.InmemDummyClient)
	if !ok {
		return fmt.Errorf("Error casting to InmemProp")
	}
	prox.SubmitTx([]byte(tx))
	return nil
}

func shutdownNodesSlice(t *testing.T, nodes []*Node, keys []uint) {
	t.Log("shutdownNodesSlice", nodes)

	for i := range keys {
		j := keys[len(keys)-1-i]

		if j >= 0 && j < uint(len(nodes)) {
			t.Logf("Shutting down node %d", j)
			nodes[j].Shutdown()
		} else {
			t.Logf("Node ")
		}

	}
}

func shutdownNodes(nodes []*Node) {
	for _, n := range nodes {
		n.Shutdown()
	}
}

func checkGossip(nodes []*Node, fromBlock int, t *testing.T) {
	t.Log("checkGossip fromBlock: ", fromBlock)

	nodeBlocks := map[int][]*hg.Block{}
	for index, n := range nodes {
		blocks := []*hg.Block{}
		for i := fromBlock; i < n.core.hg.Store.LastBlockIndex(); i++ {
			block, err := n.core.hg.Store.GetBlock(i)
			if err != nil {
				t.Fatalf("Fatal checkGossip: %v ", err)
			}
			blocks = append(blocks, block)
		}
		nodeBlocks[index] = blocks
	}

	minB := len(nodeBlocks[0])
	for k := 1; k < len(nodes); k++ {
		if len(nodeBlocks[k]) < minB {
			minB = len(nodeBlocks[k])
		}
	}

	for i, block := range nodeBlocks[0][:minB] {
		for k := 1; k < len(nodes); k++ {
			oBlock := nodeBlocks[k][i]
			if !reflect.DeepEqual(block.Body, oBlock.Body) {
				t.Fatalf("Fatal checkGossip: Difference in Block %d. ###### nodes[0]: %#v ###### nodes[%d]: %#v", block.Index(), block.Body, k, oBlock.Body)
			}
		}
	}
}

func checkTimestamps(node *Node, t *testing.T) {
	lastTimestamp := int64(0)
	for i := 0; i < node.core.hg.Store.LastBlockIndex(); i++ {
		block, err := node.core.hg.Store.GetBlock(i)
		if err != nil {
			t.Fatalf("Fatal checkTimestamps: %v ", err)
		}

		if block.Timestamp() < lastTimestamp {
			t.Fatalf("not increasing timestamp: block %d (%d) <= %d", i, block.Timestamp(), lastTimestamp)
		}

		lastTimestamp = block.Timestamp()
	}
}

func drawGraphs(nodes []*Node, t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)
	for _, n := range nodes {
		graph := NewGraph(n)

		n.coreLock.Lock()
		info, err := graph.GetInfos()
		n.coreLock.Unlock()

		if err != nil {
			t.Logf("ERROR drawing graph: %s", err)
			continue
		}

		jinfo, err := json.Marshal(info)
		if err != nil {
			t.Log(err)
		}

		err = ioutil.WriteFile(fmt.Sprintf("test_data/info%d", n.core.validator.ID()), jinfo, 0644)
		if err != nil {
			t.Log(err)
		}
	}
}

func deleteStores(nodes []*Node, t *testing.T) {
	for _, n := range nodes {
		if err := os.RemoveAll(n.core.hg.Store.StorePath()); err != nil {
			t.Error("Fatal Error deleteStores", err)
			t.Fatal(err)
		}
	}
}

func getCommittedTransactions(n *Node) ([][]byte, error) {
	InmemAppProxy, ok := n.proxy.(*dummy.InmemDummyClient)
	if !ok {
		return nil, fmt.Errorf("Error casting to InmemProp")
	}
	res := InmemAppProxy.GetCommittedTransactions()
	return res, nil
}

//Function to build a nice representation of the peerlist and put it in the logs.
func logPeerList(t testing.TB, peers *peers.PeerSet, msg string) {
	comma := ""
	iplist := ""

	for i, p := range peers.Peers {
		iplist += comma + strconv.Itoa(i) + ": " + p.NetAddr
		comma = ", "
	}

	t.Log(msg, iplist)
}

func logNodeList(t testing.TB, nodes []*Node, msg string) {
	comma := ""
	iplist := ""

	for i, p := range nodes {
		iplist += comma + strconv.Itoa(i) + ": " + fmt.Sprintf("%x", p.core.validator.ID())
		comma = ", "
	}

	t.Log(msg, iplist)
}

func peerDifference(slice1 []*peers.Peer, slice2 []*peers.Peer) []string {
	var diff []string

	for _, s1 := range slice1 {
		found := false
		for _, s2 := range slice2 {
			if s1.PubKeyHex == s2.PubKeyHex {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, s1.PubKeyHex)
		}
	}

	return diff
}

func checkFrames(nodes []*Node, fromRound int, t *testing.T) {
	t.Log("checkFrames fromRound: ", fromRound)

	var maxFrames []int

	for _, k := range nodes {
		maxFrames = append(maxFrames, k.core.hg.Store.LastRound())
	}

	t.Logf("Max Frame Rounds %#v", maxFrames)

	n := nodes[0]

	for i := fromRound; i < n.core.hg.Store.LastRound(); i++ {
		for j, n2 := range nodes {
			if n == n2 {
				continue
			}
			frame, err := n.core.hg.Store.GetFrame(i)
			if err != nil {
				t.Log("Frame Load Error", err)
				continue
			}
			frame2, err2 := n2.core.hg.Store.GetFrame(i)
			if err2 != nil {
				t.Log("Frame Load Error node2", err2)
				continue
			}

			if !reflect.DeepEqual(frame.Peers, frame2.Peers) {
				// We have a difference.
				in1Only := peerDifference(frame.Peers, frame2.Peers)
				innOnly := peerDifference(frame2.Peers, frame.Peers)

				if in1Only != nil {
					t.Logf("Frame %d: In Node 0 only not node %d, %#v", i, j, in1Only)
				}
				if innOnly != nil {
					t.Logf("Frame %d: In Node %d only not node 0, %#v", i, j, innOnly)
				}
			}
		}
	}
}
