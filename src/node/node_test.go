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
	bkeys "github.com/mosaicnetworks/babble/src/crypto/keys"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	"github.com/mosaicnetworks/babble/src/peers"
	dummy "github.com/mosaicnetworks/babble/src/proxy/dummy"
	"github.com/sirupsen/logrus"
)

var ip = 9990

func TestAddTransaction(t *testing.T) {
	keys, p := initPeers(t, 2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, config.TCPTimeout, config.JoinTimeout, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	peer0Proxy := dummy.NewInmemDummyClient(testLogger)
	defer peer0Trans.Close()

	genesisPeerSet := clonePeerSet(t, p.Peers)

	node0 := NewNode(config,
		NewValidator(keys[0], peers[0].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer0Trans,
		peer0Proxy)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, config.TCPTimeout, config.JoinTimeout, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	peer1Proxy := dummy.NewInmemDummyClient(testLogger)
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t),
		NewValidator(keys[1], peers[1].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer1Trans,
		peer1Proxy)
	node1.Init()

	node1.RunAsync(false)
	//Submit a Tx to node0

	message := "Hello World!"
	peer0Proxy.SubmitTx([]byte(message))

	//simulate a SyncRequest from node0 to node1

	node0KnownEvents := node0.core.KnownEvents()
	args := net.SyncRequest{
		FromID: node0.validator.ID(),
		Known:  node0KnownEvents,
	}

	var out net.SyncResponse
	if err := peer0Trans.Sync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatal(err)
	}

	if err := node0.sync(peers[1].ID(), out.Events); err != nil {
		t.Fatal(err)
	}

	//check the Tx was removed from the transactionPool and added to the new Head

	if l := len(node0.core.transactionPool); l > 0 {
		t.Fatalf("node0's transactionPool should have 0 elements, not %d\n", l)
	}

	node0Head, _ := node0.core.GetHead()
	if l := len(node0Head.Transactions()); l != 1 {
		t.Fatalf("node0's Head should have 1 element, not %d\n", l)
	}

	if m := string(node0Head.Transactions()[0]); m != message {
		t.Fatalf("Transaction message should be '%s' not, not %s\n", message, m)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestGossip(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 100000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
	//defer drawGraphs(nodes, t)

	target := 50
	err := gossip(nodes, target, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes, 0, t)
}

func TestMissingNodeGossip(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
	//defer drawGraphs(nodes, t)

	err := gossip(nodes[1:], 10, true, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes[1:], 0, t)
}

func TestSyncLimit(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	err := gossip(nodes, 10, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	//create fake node[0] known to artificially reach SyncLimit
	node0KnownEvents := nodes[0].core.KnownEvents()
	for k := range node0KnownEvents {
		node0KnownEvents[k] = 0
	}

	args := net.SyncRequest{
		FromID: nodes[0].validator.ID(),
		Known:  node0KnownEvents,
	}
	expectedResp := net.SyncResponse{
		FromID:    nodes[1].validator.ID(),
		SyncLimit: true,
	}

	var out net.SyncResponse
	if err := nodes[0].trans.Sync(nodes[1].trans.LocalAddr(), &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.FromID != out.FromID {
		t.Fatalf("SyncResponse.FromID should be %d, not %d", expectedResp.FromID, out.FromID)
	}
	if expectedResp.SyncLimit != true {
		t.Fatal("SyncResponse.SyncLimit should be true")
	}
}

func TestFastForward(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	target := 20
	err := gossip(nodes[1:], target, false, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	err = nodes[0].fastForward()
	if err != nil {
		t.Fatalf("Error FastForwarding: %s", err)
	}

	lbi := nodes[0].core.GetLastBlockIndex()
	if lbi <= 0 {
		t.Fatalf("LastBlockIndex is too low: %d", lbi)
	}

	sBlock, err := nodes[0].GetBlock(lbi)
	if err != nil {
		t.Fatalf("Error retrieving latest Block from reset hashgraph: %v", err)
	}

	expectedBlock, err := nodes[1].GetBlock(lbi)
	if err != nil {
		t.Fatalf("Failed to retrieve block %d from node1: %v", lbi, err)
	}

	if !reflect.DeepEqual(sBlock.Body, expectedBlock.Body) {
		t.Fatalf("Blocks defer")
	}
}

func TestCatchUp(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	/*
		We don't initialize the first node; it will stay passive during the
		first part of the test, otherwise requests would be queued in its TCP
		socket, even if it is not running. This is because the socket is setup
		to listen regardless of whether a node is running or not. We should
		probably change this at some point.
	*/
	normalNodes := initNodes(keys[1:], peers, genesisPeerSet, 1000000, 100, 5, true, "inmem", 50*time.Millisecond, logger, t)
	defer shutdownNodes(normalNodes)
	//defer drawGraphs(normalNodes, t)

	target := 10
	err := gossip(normalNodes, target, false, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(normalNodes, 0, t)

	node0 := newNode(peers.Peers[0], keys[0], peers, genesisPeerSet, 1000000, 100, 5, true, "inmem", 10*time.Millisecond, logger, t)
	defer node0.Shutdown()

	//Run parallel routine to check node0 eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for node0 to enter CatchingUp state")
			default:
			}
			if node0.getState() == CatchingUp {
				break
			}
		}
	}()

	node0.RunAsync(true)

	nodes := append(normalNodes, node0)

	//Gossip some more with all nodes
	newTarget := target + 20
	err = bombardAndWait(nodes, newTarget, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := node0.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestFastSync(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 100000, 400, 5, true, "inmem", 10*time.Millisecond, logger, t) //make cache high to draw graphs
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	node0 := nodes[0]
	node0.Shutdown()

	secondTarget := target + 30
	err = bombardAndWait(nodes[1:], secondTarget, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes[1:], 0, t)

	//Can't re-run it; have to reinstantiate a new node.
	node0 = recycleNode(node0, logger, t)
	nodes[0] = node0

	//Run parallel routine to check node0 eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for node0 to enter CatchingUp state")
			default:
			}
			if node0.getState() == CatchingUp {
				break
			}
		}
	}()

	node0.RunAsync(true)

	//Gossip some more
	thirdTarget := secondTarget + 50
	err = bombardAndWait(nodes, thirdTarget, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := node0.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestShutdown(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
	runNodes(nodes, false)

	nodes[0].Shutdown()

	err := nodes[1].gossip(peers.Peers[0], nil)
	if err == nil {
		t.Fatal("Expected Timeout Error")
	}

	nodes[1].Shutdown()
}

func TestBootstrapAllNodes(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	//create a first network with BadgerStore and wait till it reaches 10 blocks
	//before shutting it down
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)
	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 10, true, "badger", 6*time.Millisecond, logger, t)

	err := gossip(nodes, 10, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	//Now try to recreate a network from the databases created in the first step
	//and advance it to 20 blocks
	newNodes := recycleNodes(nodes, logger, t)

	err = gossip(newNodes, 20, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(newNodes, 0, t)

	//Check that both networks did not have completely different consensus events
	checkGossip([]*Node{nodes[0], newNodes[0]}, 0, t)
}

func BenchmarkGossip(b *testing.B) {
	logger := common.NewTestLogger(b)
	for n := 0; n < b.N; n++ {
		keys, peers := initPeers(b, 4)

		genesisPeerSet := clonePeerSet(b, peers.Peers)

		nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, b)
		gossip(nodes, 50, true, 3*time.Second)
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
	logger *logrus.Logger,
	t testing.TB) *Node {

	conf := NewConfig(
		heartbeatTimeout,
		time.Second,
		joinTimeoutSeconds*time.Second,
		cacheSize,
		syncLimit,
		enableSyncLimit,
		logger,
	)

	t.Logf("Starting node on %s", peer.NetAddr)

	trans, err := net.NewTCPTransport(peer.NetAddr,
		nil, 2, conf.TCPTimeout, conf.JoinTimeout, logger)
	if err != nil {
		t.Fatalf("failed to create transport for peer %d: %s", peer.ID(), err)
	}

	var store hg.Store
	switch storeType {
	case "badger":
		path, _ := ioutil.TempDir("test_data", "badger")
		store, err = hg.NewBadgerStore(conf.CacheSize, path)
		if err != nil {
			t.Fatalf("failed to create BadgerStore for peer %d: %s", peer.ID(), err)
		}
	case "inmem":
		store = hg.NewInmemStore(conf.CacheSize)
	}

	prox := dummy.NewInmemDummyClient(logger)
	node := NewNode(conf,
		NewValidator(k, peer.Moniker),
		peers,
		genesisPeers,
		store,
		trans,
		prox)

	if err := node.Init(); err != nil {
		t.Fatalf("failed to initialize node%d: %s", peer.ID(), err)
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
	logger *logrus.Logger,
	t testing.TB) []*Node {

	nodes := []*Node{}

	for _, k := range keys {
		pubKey := bkeys.PublicKeyHex(&k.PublicKey)

		peer, ok := peers.ByPubKey[pubKey]
		if !ok {
			t.Fatalf("Peer not found")
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
			logger,
			t)

		nodes = append(nodes, node)
	}
	return nodes
}

func recycleNodes(oldNodes []*Node, logger *logrus.Logger, t *testing.T) []*Node {
	newNodes := []*Node{}
	for _, oldNode := range oldNodes {
		newNode := recycleNode(oldNode, logger, t)
		newNodes = append(newNodes, newNode)
	}
	return newNodes
}

func recycleNode(oldNode *Node, logger *logrus.Logger, t *testing.T) *Node {
	conf := oldNode.conf
	key := oldNode.validator.Key
	moniker := oldNode.validator.Moniker
	peers := oldNode.core.peers
	genesisPeerSet := oldNode.core.genesisPeers

	var store hg.Store
	var err error
	if _, ok := oldNode.core.hg.Store.(*hg.BadgerStore); ok {
		store, err = hg.NewBadgerStore(conf.CacheSize, oldNode.core.hg.Store.StorePath())
		if err != nil {
			t.Fatal(err)
		}
	} else {
		store = hg.NewInmemStore(conf.CacheSize)
	}

	trans, err := net.NewTCPTransport(oldNode.trans.LocalAddr(),
		nil, 2, conf.TCPTimeout, conf.JoinTimeout, logger)
	if err != nil {
		t.Fatal(err)
	}
	prox := dummy.NewInmemDummyClient(logger)

	conf.Bootstrap = true

	newNode := NewNode(conf, NewValidator(key, moniker), peers, genesisPeerSet, store, trans, prox)

	if err := newNode.Init(); err != nil {
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

func gossip(nodes []*Node, target int, shutdown bool, timeout time.Duration) error {
	runNodes(nodes, true)
	err := bombardAndWait(nodes, target, timeout)
	if err != nil {
		return err
	}
	if shutdown {
		shutdownNodes(nodes)
	}
	return nil
}

func bombardAndWait(nodes []*Node, target int, timeout time.Duration) error {
	//send a lot of random transactions to the nodes
	quit := make(chan struct{})
	makeRandomTransactions(nodes, quit)

	//wait until all nodes reach at least block 'target'
	stopper := time.After(timeout)
	for {
		select {
		case <-stopper:
			return fmt.Errorf("TIMEOUT")
		default:
		}
		time.Sleep(10 * time.Millisecond)
		done := true
		for _, n := range nodes {
			ce := n.core.GetLastBlockIndex()
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
			break
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
				t.Fatalf("checkGossip: %v ", err)
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
				t.Fatalf("checkGossip: Difference in Block %d. ###### nodes[0]: %v ###### nodes[%d]: %v", block.Index(), block.Body, k, oBlock.Body)
			}
		}
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

		err = ioutil.WriteFile(fmt.Sprintf("test_data/info%d", n.ID()), jinfo, 0644)
		if err != nil {
			t.Log(err)
		}
	}
}

func deleteStores(nodes []*Node, t *testing.T) {
	for _, n := range nodes {
		if err := os.RemoveAll(n.core.hg.Store.StorePath()); err != nil {
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
		iplist += comma + strconv.Itoa(i) + ": " + fmt.Sprintf("%x", p.validator.ID())
		comma = ", "
	}

	t.Log(msg, iplist)
}
