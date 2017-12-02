package node

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/babbleio/babble/common"
	"github.com/babbleio/babble/crypto"
	hg "github.com/babbleio/babble/hashgraph"
	"github.com/babbleio/babble/net"
	aproxy "github.com/babbleio/babble/proxy/app"
)

var ip = 9990

func initPeers(n int) ([]*ecdsa.PrivateKey, []net.Peer, map[string]int) {
	keys := []*ecdsa.PrivateKey{}
	peers := []net.Peer{}

	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		keys = append(keys, key)
		peers = append(peers, net.Peer{
			NetAddr:   fmt.Sprintf("127.0.0.1:%d", ip),
			PubKeyHex: fmt.Sprintf("0x%X", crypto.FromECDSAPub(&keys[i].PublicKey)),
		})
		ip++
	}

	sort.Sort(net.ByPubKey(peers))
	pmap := make(map[string]int)
	for i, p := range peers {
		pmap[p.PubKeyHex] = i
	}

	return keys, peers, pmap
}

func TestProcessSync(t *testing.T) {
	keys, peers, pmap := initPeers(2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(config, pmap[peers[0].PubKeyHex], keys[0], peers,
		hg.NewInmemStore(pmap, config.CacheSize),
		peer0Trans,
		aproxy.NewInmemAppProxy(testLogger))
	node0.Init(false)

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(config, pmap[peers[1].PubKeyHex], keys[1], peers,
		hg.NewInmemStore(pmap, config.CacheSize),
		peer1Trans,
		aproxy.NewInmemAppProxy(testLogger))
	node1.Init(false)

	node1.RunAsync(false)

	//Manually prepare SyncRequest and expected SyncResponse

	node0Known := node0.core.Known()
	node1Known := node1.core.Known()

	unknown, err := node1.core.Diff(node0Known)
	if err != nil {
		t.Fatal(err)
	}

	unknownWire, err := node1.core.ToWire(unknown)
	if err != nil {
		t.Fatal(err)
	}

	args := net.SyncRequest{
		From:  node0.localAddr,
		Known: node0Known,
	}
	expectedResp := net.SyncResponse{
		From:   node1.localAddr,
		Events: unknownWire,
		Known:  node1Known,
	}

	//Make actual SyncRequest and check SyncResponse

	var out net.SyncResponse
	if err := peer0Trans.Sync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.From != out.From {
		t.Fatalf("SyncResponse.From should be %s, not %s", expectedResp.From, out.From)
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
		t.Fatalf("SyncResponse.Known should be %#v, not %#v", expectedResp.Known, out.Known)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestProcessEagerSync(t *testing.T) {
	keys, peers, pmap := initPeers(2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), pmap[peers[0].PubKeyHex], keys[0], peers,
		hg.NewInmemStore(pmap, config.CacheSize),
		peer0Trans,
		aproxy.NewInmemAppProxy(testLogger))
	node0.Init(false)

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), pmap[peers[1].PubKeyHex], keys[1], peers,
		hg.NewInmemStore(pmap, config.CacheSize),
		peer1Trans,
		aproxy.NewInmemAppProxy(testLogger))
	node1.Init(false)

	node1.RunAsync(false)

	//Manually prepare EagerSyncRequest and expected EagerSyncResponse

	node1Known := node1.core.Known()

	unknown, err := node0.core.Diff(node1Known)
	if err != nil {
		t.Fatal(err)
	}

	unknownWire, err := node0.core.ToWire(unknown)
	if err != nil {
		t.Fatal(err)
	}

	args := net.EagerSyncRequest{
		From:   node0.localAddr,
		Events: unknownWire,
	}
	expectedResp := net.EagerSyncResponse{
		From:    node1.localAddr,
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
	keys, peers, pmap := initPeers(2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()
	peer0Proxy := aproxy.NewInmemAppProxy(testLogger)

	node0 := NewNode(TestConfig(t), pmap[peers[0].PubKeyHex], keys[0], peers,
		hg.NewInmemStore(pmap, config.CacheSize),
		peer0Trans,
		peer0Proxy)
	node0.Init(false)

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()
	peer1Proxy := aproxy.NewInmemAppProxy(testLogger)

	node1 := NewNode(TestConfig(t), pmap[peers[1].PubKeyHex], keys[1], peers,
		hg.NewInmemStore(pmap, config.CacheSize),
		peer1Trans,
		peer1Proxy)
	node1.Init(false)

	node1.RunAsync(false)

	//Submit a Tx to node0

	message := "Hello World!"
	peer0Proxy.SubmitTx([]byte(message))

	//simulate a SyncRequest from node0 to node1

	node0Known := node0.core.Known()
	args := net.SyncRequest{
		From:  node0.localAddr,
		Known: node0Known,
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

func initNodes(n, cacheSize, syncLimit int, storeType string,
	logger *logrus.Logger, t testing.TB) ([]*ecdsa.PrivateKey, []*Node) {

	keys, peers, pmap := initPeers(n)
	nodes := []*Node{}
	proxies := []*aproxy.InmemAppProxy{}
	for i := 0; i < len(peers); i++ {
		conf := NewConfig(5*time.Millisecond, time.Second, cacheSize, syncLimit,
			storeType, fmt.Sprintf("test_data/db_%d", i), logger)

		trans, err := net.NewTCPTransport(peers[i].NetAddr,
			nil, 2, time.Second, logger)
		if err != nil {
			t.Fatalf("failed to create transport for peer %d: %s", i, err)
		}
		var store hg.Store
		switch storeType {
		case "badger":
			store, err = hg.NewBadgerStore(pmap, conf.CacheSize, conf.StorePath)
			if err != nil {
				t.Fatalf("failed to create BadgerStore for peer %d: %s", i, err)
			}
		case "inmem":
			store = hg.NewInmemStore(pmap, conf.CacheSize)
		}
		prox := aproxy.NewInmemAppProxy(logger)
		node := NewNode(conf, pmap[peers[i].PubKeyHex], keys[i], peers,
			store,
			trans,
			prox)
		if err := node.Init(false); err != nil {
			t.Fatalf("failed to initialize node%d: %s", i, err)
		}
		nodes = append(nodes, node)
		proxies = append(proxies, prox)
	}
	return keys, nodes
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
	peers := oldNode.peerSelector.Peers()
	store, err := hg.LoadBadgerStore(conf.CacheSize, conf.StorePath)
	if err != nil {
		t.Fatal(err)
	}
	trans, err := net.NewTCPTransport(oldNode.localAddr,
		nil, 2, time.Second, logger)
	if err != nil {
		t.Fatal(err)
	}
	prox := aproxy.NewInmemAppProxy(logger)

	newNode := NewNode(conf, id, key, peers, store, trans, prox)

	if err := newNode.Init(true); err != nil {
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

func deleteStores(nodes []*Node, t *testing.T) {
	for _, n := range nodes {
		if err := os.RemoveAll(n.conf.StorePath); err != nil {
			t.Fatal(err)
		}
	}
}

func getCommittedTransactions(n *Node) ([][]byte, error) {
	InmemAppProxy, ok := n.proxy.(*aproxy.InmemAppProxy)
	if !ok {
		return nil, fmt.Errorf("Error casting to InmemProp")
	}
	res := InmemAppProxy.GetCommittedTransactions()
	return res, nil
}

func TestGossip(t *testing.T) {
	logger := common.NewTestLogger(t)

	_, nodes := initNodes(4, 1000, 1000, "inmem", logger, t)

	err := gossip(nodes, 50, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes, t)
}

func TestMissingNodeGossip(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(4, 1000, 1000, "inmem", logger, t)
	defer shutdownNodes(nodes)

	err := gossip(nodes[1:], 10, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes[1:], t)
}

func TestSyncLimit(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(4, 1000, 300, "inmem", logger, t)

	err := gossip(nodes, 10, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer shutdownNodes(nodes)

	//create fake node[0] known to artificially reach SyncLimit
	node0Known := nodes[0].core.Known()
	for k := range node0Known {
		node0Known[k] = 0
	}

	args := net.SyncRequest{
		From:  nodes[0].localAddr,
		Known: node0Known,
	}
	expectedResp := net.SyncResponse{
		From:      nodes[1].localAddr,
		SyncLimit: true,
	}

	var out net.SyncResponse
	if err := nodes[0].trans.Sync(nodes[1].localAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.From != out.From {
		t.Fatalf("SyncResponse.From should be %s, not %s", expectedResp.From, out.From)
	}
	if expectedResp.SyncLimit != true {
		t.Fatal("SyncResponse.SyncLimit should be true")
	}
}

func TestShutdown(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(2, 1000, 1000, "inmem", logger, t)

	runNodes(nodes, false)

	nodes[0].Shutdown()

	err := nodes[1].gossip(nodes[0].localAddr)
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
	_, nodes := initNodes(4, 10000, 1000, "badger", logger, t)
	err := gossip(nodes, 10, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, t)
	shutdownNodes(nodes)

	//Now try to recreate a network from the databases created in the first step
	//and advance it to 20 consensus rounds
	newNodes := recycleNodes(nodes, logger, t)
	err = gossip(newNodes, 20, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(newNodes, t)
	shutdownNodes(newNodes)

	//Check that both networks did not have completely different consensus events
	checkGossip([]*Node{nodes[0], newNodes[0]}, t)
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

	//wait until all nodes have at least 'target' rounds
	stopper := time.After(timeout)
	for {
		select {
		case <-stopper:
			return fmt.Errorf("timeout")
		default:
		}
		time.Sleep(10 * time.Millisecond)
		done := true
		for _, n := range nodes {
			ce := n.core.GetLastConsensusRoundIndex()
			if ce == nil || *ce < target {
				done = false
				break
			}
		}
		if done {
			break
		}
	}
	close(quit)
	return nil
}

func checkGossip(nodes []*Node, t *testing.T) {
	consEvents := map[int][]string{}
	consTransactions := map[int][][]byte{}
	for _, n := range nodes {
		consEvents[n.id] = n.core.GetConsensusEvents()
		nodeTxs, err := getCommittedTransactions(n)
		if err != nil {
			t.Fatal(err)
		}
		consTransactions[n.id] = nodeTxs
	}

	minE := len(consEvents[0])
	minT := len(consTransactions[0])
	for k := 1; k < len(nodes); k++ {
		if len(consEvents[k]) < minE {
			minE = len(consEvents[k])
		}
		if len(consTransactions[k]) < minT {
			minT = len(consTransactions[k])
		}
	}

	problem := false
	t.Logf("min consensus events: %d", minE)
	for i, e := range consEvents[0][0:minE] {
		for j := range nodes[1:len(nodes)] {
			if f := consEvents[j][i]; f != e {
				er := nodes[0].core.hg.Round(e)
				err := nodes[0].core.hg.RoundReceived(e)
				fr := nodes[j].core.hg.Round(f)
				frr := nodes[j].core.hg.RoundReceived(f)
				t.Logf(
					"nodes[%d].Consensus[%d] (%s, Round %d, Received %d) and nodes[0].Consensus[%d] (%s, Round %d, Received %d) are not equal",
					j, i, e[:6], er, err, i, f[:6], fr, frr)
				problem = true
			}
		}
	}
	if problem {
		t.Fatal()
	}

	t.Logf("min consensus transactions: %d", minT)
	for i, tx := range consTransactions[0][:minT] {
		for k := range nodes[1:len(nodes)] {
			if ot := string(consTransactions[k][i]); ot != string(tx) {
				t.Fatalf("nodes[%d].ConsensusTransactions[%d] should be '%s' not '%s'", k, i, string(tx), ot)
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
	prox, ok := n.proxy.(*aproxy.InmemAppProxy)
	if !ok {
		return fmt.Errorf("Error casting to InmemProp")
	}
	prox.SubmitTx([]byte(tx))
	return nil
}

func BenchmarkGossip(b *testing.B) {
	logger := common.NewBenchmarkLogger(b)
	for n := 0; n < b.N; n++ {
		_, nodes := initNodes(3, 1000, 1000, "inmem", logger, b)
		gossip(nodes, 5, true, 3*time.Second)
	}
}
