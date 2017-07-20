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
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/babbleio/babble/common"
	"github.com/babbleio/babble/crypto"
	"github.com/babbleio/babble/net"
	aproxy "github.com/babbleio/babble/proxy/app"
)

var ip = 9990

func initPeers() ([]*ecdsa.PrivateKey, []net.Peer) {
	keys := []*ecdsa.PrivateKey{}
	peers := []net.Peer{}

	n := 3
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
	return keys, peers
}

func TestProcessSync(t *testing.T) {
	keys, peers := initPeers()
	testLogger := common.NewTestLogger(t)

	//Start two nodes

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans, aproxy.NewInmemAppProxy(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans, aproxy.NewInmemAppProxy(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare SyncRequest and expected SyncResponse

	node0Known := node0.core.Known()
	node1Known := node1.core.Known()

	head, unknown, err := node1.core.Diff(node0Known)
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
		Head:   head,
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

	if expectedResp.Head != out.Head {
		t.Fatalf("SyncResponse.Head should be %s, not %s", expectedResp.Head, out.Head)
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
	keys, peers := initPeers()
	testLogger := common.NewTestLogger(t)

	//Start two nodes

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans, aproxy.NewInmemAppProxy(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans, aproxy.NewInmemAppProxy(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare EagerSyncRequest and expected EagerSyncResponse

	node1Known := node1.core.Known()

	head, unknown, err := node0.core.Diff(node1Known)
	if err != nil {
		t.Fatal(err)
	}

	unknownWire, err := node0.core.ToWire(unknown)
	if err != nil {
		t.Fatal(err)
	}

	args := net.EagerSyncRequest{
		From:   node0.localAddr,
		Head:   head,
		Events: unknownWire,
	}
	expectedResp := net.EagerSyncResponse{
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
	keys, peers := initPeers()
	testLogger := common.NewTestLogger(t)

	//Start two nodes

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()
	peer0Proxy := aproxy.NewInmemAppProxy(testLogger)

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans, peer0Proxy)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()
	peer1Proxy := aproxy.NewInmemAppProxy(testLogger)

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans, peer1Proxy)
	node1.Init()

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

	if err := node0.sync(out.Head, out.Events); err != nil {
		t.Fatal(err)
	}

	//check the Tx was removed from the transactionPool and added to the new Head

	if l := len(node0.transactionPool); l > 0 {
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

func initNodes(logger *logrus.Logger) ([]*ecdsa.PrivateKey, []*Node) {
	conf := NewConfig(5*time.Millisecond, time.Second, 1000, logger)

	keys, peers := initPeers()
	nodes := []*Node{}
	proxies := []*aproxy.InmemAppProxy{}
	for i := 0; i < len(peers); i++ {
		trans, err := net.NewTCPTransport(peers[i].NetAddr,
			nil, 2, time.Second, logger)
		if err != nil {
			logger.Panicf("failed to create transport for peer %d: %s\n", i, err.Error())
		}
		prox := aproxy.NewInmemAppProxy(logger)
		node := NewNode(conf, keys[i], peers, trans, prox)
		node.Init()
		nodes = append(nodes, &node)
		proxies = append(proxies, prox)
	}
	return keys, nodes
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
		n.trans.Close()
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
	_, nodes := initNodes(logger)

	gossip(nodes, 100)

	consEvents := [][]string{}
	consTransactions := [][][]byte{}
	for _, n := range nodes {
		consEvents = append(consEvents, n.core.GetConsensusEvents())
		nodeTxs, err := getCommittedTransactions(n)
		if err != nil {
			t.Fatal(err)
		}
		consTransactions = append(consTransactions, nodeTxs)
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

	t.Logf("min consensus events: %d", minE)
	for i, e := range consEvents[0][0:minE] {
		for j := range nodes[1:len(nodes)] {
			if consEvents[j][i] != e {
				t.Fatalf("nodes[%d].Consensus[%d] and nodes[0].Consensus[%d] are not equal", j, i, i)
			}
		}
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

func gossip(nodes []*Node, target int) {
	runNodes(nodes, true)
	quit := make(chan int)
	makeRandomTransactions(nodes, quit)

	//wait until all nodes have at least 'target' consensus events
	for {
		time.Sleep(10 * time.Millisecond)
		done := true
		for _, n := range nodes {
			ce := n.core.GetConsensusEventsCount()
			if ce < target {
				done = false
				break
			}
		}
		if done {
			break
		}
	}

	close(quit)
	shutdownNodes(nodes)
}

func submitTransaction(n *Node, tx []byte) error {
	prox, ok := n.proxy.(*aproxy.InmemAppProxy)
	if !ok {
		return fmt.Errorf("Error casting to InmemProp")
	}
	prox.SubmitTx([]byte(tx))
	return nil
}

func makeRandomTransactions(nodes []*Node, quit chan int) {
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

func BenchmarkGossip(b *testing.B) {
	logger := common.NewBenchmarkLogger(b)
	for n := 0; n < b.N; n++ {
		_, nodes := initNodes(logger)
		gossip(nodes, 5)
	}
}
