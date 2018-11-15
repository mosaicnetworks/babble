package node

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	"github.com/mosaicnetworks/babble/src/peers"
	dummy "github.com/mosaicnetworks/babble/src/proxy/dummy"
	"github.com/sirupsen/logrus"
)

var ip = 9990

func initPeers(n int) ([]*ecdsa.PrivateKey, *peers.PeerSet) {
	keys := []*ecdsa.PrivateKey{}
	pirs := []*peers.Peer{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		keys = append(keys, key)
		peer := peers.NewPeer(
			fmt.Sprintf("0x%X", crypto.FromECDSAPub(&keys[i].PublicKey)),
			fmt.Sprintf("127.0.0.1:%d", ip),
		)
		pirs = append(pirs, peer)
		ip++
	}

	peerSet := peers.NewPeerSet(pirs)

	return keys, peerSet
}

func TestProcessSync(t *testing.T) {
	keys, p := initPeers(2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(config, peers[0].ID, keys[0], p,
		hg.NewInmemStore(p, config.CacheSize),
		peer0Trans,
		dummy.NewInmemDummyClient(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(config, peers[1].ID, keys[1], p,
		hg.NewInmemStore(p, config.CacheSize),
		peer1Trans,
		dummy.NewInmemDummyClient(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare SyncRequest and expected SyncResponse

	node0KnownEvents := node0.core.KnownEvents()
	node1KnownEvents := node1.core.KnownEvents()

	unknownEvents, err := node1.core.EventDiff(node0KnownEvents)
	if err != nil {
		t.Fatal(err)
	}

	unknownWireEvents, err := node1.core.ToWire(unknownEvents)
	if err != nil {
		t.Fatal(err)
	}

	args := net.SyncRequest{
		FromID: node0.id,
		Known:  node0KnownEvents,
	}
	expectedResp := net.SyncResponse{
		FromID: node1.id,
		Events: unknownWireEvents,
		Known:  node1KnownEvents,
	}

	//Make actual SyncRequest and check SyncResponse

	var out net.SyncResponse
	if err := peer0Trans.Sync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.FromID != out.FromID {
		t.Fatalf("SyncResponse.FromID should be %d, not %d", expectedResp.FromID, out.FromID)
	}

	if l := len(out.Events); l != len(expectedResp.Events) {
		t.Fatalf("SyncResponse.Events should contain %d items, not %d",
			len(expectedResp.Events), l)
	}

	for i, e := range expectedResp.Events {
		ex := out.Events[i]
		if !reflect.DeepEqual(e.Body, ex.Body) {
			t.Fatalf("SyncResponse.Events[%d] should be %v, not %v", i, e.Body,
				ex.Body)
		}
	}

	if !reflect.DeepEqual(expectedResp.Known, out.Known) {
		t.Fatalf("SyncResponse.KnownEvents should be %#v, not %#v",
			expectedResp.Known, out.Known)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestProcessEagerSync(t *testing.T) {
	keys, p := initPeers(2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(config, peers[0].ID, keys[0], p,
		hg.NewInmemStore(p, config.CacheSize),
		peer0Trans,
		dummy.NewInmemDummyClient(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(config, peers[1].ID, keys[1], p,
		hg.NewInmemStore(p, config.CacheSize),
		peer1Trans,
		dummy.NewInmemDummyClient(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare EagerSyncRequest and expected EagerSyncResponse

	node1KnownEvents := node1.core.KnownEvents()

	unknownEvents, err := node0.core.EventDiff(node1KnownEvents)
	if err != nil {
		t.Fatal(err)
	}

	unknownWireEvents, err := node0.core.ToWire(unknownEvents)
	if err != nil {
		t.Fatal(err)
	}

	args := net.EagerSyncRequest{
		FromID: node0.id,
		Events: unknownWireEvents,
	}
	expectedResp := net.EagerSyncResponse{
		FromID:  node1.id,
		Success: true,
	}

	//Make actual EagerSyncRequest and check EagerSyncResponse

	var out net.EagerSyncResponse
	if err := peer0Trans.EagerSync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.Success != out.Success {
		t.Fatalf("EagerSyncResponse.Sucess should be %v, not %v", expectedResp.Success, out.Success)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestAddTransaction(t *testing.T) {
	keys, p := initPeers(2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	peer0Proxy := dummy.NewInmemDummyClient(testLogger)
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), peers[0].ID, keys[0], p,
		hg.NewInmemStore(p, config.CacheSize),
		peer0Trans,
		peer0Proxy)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	peer1Proxy := dummy.NewInmemDummyClient(testLogger)
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), peers[1].ID, keys[1], p,
		hg.NewInmemStore(p, config.CacheSize),
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
		FromID: node0.id,
		Known:  node0KnownEvents,
	}

	var out net.SyncResponse
	if err := peer0Trans.Sync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatal(err)
	}

	if err := node0.sync(out.Events); err != nil {
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

func initNodes(keys []*ecdsa.PrivateKey,
	peers *peers.PeerSet,
	cacheSize,
	syncLimit int,
	storeType string,
	logger *logrus.Logger,
	t testing.TB) []*Node {

	nodes := []*Node{}

	for _, k := range keys {
		key := fmt.Sprintf("0x%X", crypto.FromECDSAPub(&k.PublicKey))
		peer := peers.ByPubKey[key]
		id := peer.ID

		conf := NewConfig(
			5*time.Millisecond,
			time.Second,
			cacheSize,
			syncLimit,
			logger,
		)

		trans, err := net.NewTCPTransport(peer.NetAddr,
			nil, 2, time.Second, logger)
		if err != nil {
			t.Fatalf("failed to create transport for peer %d: %s", id, err)
		}
		var store hg.Store
		switch storeType {
		case "badger":
			path, _ := ioutil.TempDir("", "badger")
			store, err = hg.NewBadgerStore(peers, conf.CacheSize, path)
			if err != nil {
				t.Fatalf("failed to create BadgerStore for peer %d: %s", id, err)
			}
		case "inmem":
			store = hg.NewInmemStore(peers, conf.CacheSize)
		}
		prox := dummy.NewInmemDummyClient(logger)
		node := NewNode(conf,
			id,
			k,
			peers,
			store,
			trans,
			prox)

		if err := node.Init(); err != nil {
			t.Fatalf("failed to initialize node%d: %s", id, err)
		}
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
	id := oldNode.id
	key := oldNode.core.key
	peers := oldNode.core.peers

	var store hg.Store
	var err error
	if _, ok := oldNode.core.hg.Store.(*hg.BadgerStore); ok {
		store, err = hg.LoadBadgerStore(conf.CacheSize, oldNode.core.hg.Store.StorePath())
		if err != nil {
			t.Fatal(err)
		}
	} else {
		store = hg.NewInmemStore(oldNode.core.peers, conf.CacheSize)
	}

	trans, err := net.NewTCPTransport(oldNode.trans.LocalAddr(),
		nil, 2, time.Second, logger)
	if err != nil {
		t.Fatal(err)
	}
	prox := dummy.NewInmemDummyClient(logger)

	newNode := NewNode(conf, id, key, peers, store, trans, prox)

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

func shutdownNodes(nodes []*Node) {
	for _, n := range nodes {
		n.Shutdown()
	}
}

func drawGraphs(nodes []*Node, t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)
	for _, n := range nodes {
		graph := NewGraph(n)
		info, err := graph.GetInfos()
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

func TestGossip(t *testing.T) {
	logger := common.NewTestLogger(t)

	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 1000, "inmem", logger, t)

	defer drawGraphs(nodes, t)

	target := 50
	err := gossip(nodes, target, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes, 0, t)
}

func TestMissingNodeGossip(t *testing.T) {
	logger := common.NewTestLogger(t)

	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 1000, "inmem", logger, t)

	defer shutdownNodes(nodes)

	defer drawGraphs(nodes, t)

	err := gossip(nodes[1:], 10, true, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes[1:], 0, t)
}

func TestSyncLimit(t *testing.T) {

	logger := common.NewTestLogger(t)

	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 1000, "inmem", logger, t)

	err := gossip(nodes, 10, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer shutdownNodes(nodes)

	//create fake node[0] known to artificially reach SyncLimit
	node0KnownEvents := nodes[0].core.KnownEvents()
	for k := range node0KnownEvents {
		node0KnownEvents[k] = 0
	}

	args := net.SyncRequest{
		FromID: nodes[0].id,
		Known:  node0KnownEvents,
	}
	expectedResp := net.SyncResponse{
		FromID:    nodes[1].id,
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

	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 1000, "inmem", logger, t)
	defer shutdownNodes(nodes)

	target := 50
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

	//Create  config for 4 nodes
	keys, peers := initPeers(4)

	//Initialize the first 3 nodes only
	normalNodes := initNodes(keys[0:3], peers, 1000, 400, "inmem", logger, t)
	defer shutdownNodes(normalNodes)

	target := 50

	err := gossip(normalNodes, target, false, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(normalNodes, 0, t)

	node4 := initNodes(keys[3:], peers, 1000, 400, "inmem", logger, t)[0]

	//Run parallel routine to check node4 eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for node4 to enter CatchingUp state")
			default:
			}
			if node4.getState() == CatchingUp {
				break
			}
		}
	}()

	node4.RunAsync(true)
	defer node4.Shutdown()

	//Gossip some more
	nodes := append(normalNodes, node4)
	newTarget := target + 20
	err = bombardAndWait(nodes, newTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := node4.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestFastSync(t *testing.T) {
	logger := common.NewTestLogger(t)

	//Create  config for 4 nodes
	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 400, "inmem", logger, t)

	defer shutdownNodes(nodes)

	defer drawGraphs(nodes, t)

	target := 50

	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	node4 := nodes[3]
	node4.Shutdown()

	secondTarget := target + 50
	err = bombardAndWait(nodes[0:3], secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes[0:3], 0, t)

	//Can't re-run it; have to reinstantiate a new node.
	node4 = recycleNode(node4, logger, t)

	//Run parallel routine to check node4 eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for node4 to enter CatchingUp state")
			default:
			}
			if node4.getState() == CatchingUp {
				break
			}
		}
	}()

	node4.RunAsync(true)
	defer node4.Shutdown()

	nodes[3] = node4

	//Gossip some more
	thirdTarget := secondTarget + 20
	err = bombardAndWait(nodes, thirdTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := node4.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestShutdown(t *testing.T) {
	logger := common.NewTestLogger(t)

	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 1000, "inmem", logger, t)
	runNodes(nodes, false)

	nodes[0].Shutdown()

	err := nodes[1].gossip(peers.Peers[0], nil)
	if err == nil {
		t.Fatal("Expected Timeout Error")
	}

	nodes[1].Shutdown()
}

func TestBootstrapAllNodes(t *testing.T) {
	logger := common.NewTestLogger(t)

	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	//create a first network with BadgerStore and wait till it reaches 10 consensus
	//rounds before shutting it down
	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 1000, 1000, "badger", logger, t)
	err := gossip(nodes, 10, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)
	shutdownNodes(nodes)

	//Now try to recreate a network from the databases created in the first step
	//and advance it to 20 consensus rounds
	newNodes := recycleNodes(nodes, logger, t)
	err = gossip(newNodes, 20, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(newNodes, 0, t)
	shutdownNodes(newNodes)

	//Check that both networks did not have completely different consensus events
	checkGossip([]*Node{nodes[0], newNodes[0]}, 0, t)
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

	quit := make(chan struct{})
	makeRandomTransactions(nodes, quit)

	//wait until all nodes have at least 'target' blocks
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
				targetBlock, _ := n.core.hg.Store.GetBlock(target)
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

func checkGossip(nodes []*Node, fromBlock int, t *testing.T) {
	nodeBlocks := map[int][]*hg.Block{}
	for _, n := range nodes {
		blocks := []*hg.Block{}
		for i := fromBlock; i < n.core.hg.Store.LastBlockIndex(); i++ {
			block, err := n.core.hg.Store.GetBlock(i)
			if err != nil {
				t.Fatalf("checkGossip: %v ", err)
			}
			blocks = append(blocks, block)
		}
		nodeBlocks[n.id] = blocks
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

func BenchmarkGossip(b *testing.B) {
	logger := common.NewTestLogger(b)
	for n := 0; n < b.N; n++ {
		keys, peers := initPeers(4)
		nodes := initNodes(keys, peers, 1000, 1000, "inmem", logger, b)
		gossip(nodes, 50, true, 3*time.Second)
	}
}
