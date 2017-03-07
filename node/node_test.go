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
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/arrivets/babble/common"
	"github.com/arrivets/babble/crypto"
	"github.com/arrivets/babble/net"
	"github.com/arrivets/babble/proxy"
)

func initPeers() ([]*ecdsa.PrivateKey, []net.Peer) {
	keys := []*ecdsa.PrivateKey{}
	peers := []net.Peer{}

	n := 3
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		keys = append(keys, key)
		peers = append(peers, net.Peer{
			NetAddr:   fmt.Sprintf("127.0.0.1:999%d", i),
			PubKeyHex: fmt.Sprintf("0x%X", crypto.FromECDSAPub(&keys[i].PublicKey)),
		})
	}
	return keys, peers
}

func TestProcessKnown(t *testing.T) {
	keys, peers := initPeers()

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node := NewNode(TestConfig(t), keys[0], peers, peer0Trans, proxy.NewInmemProxy(common.NewTestLogger(t)))
	node.Init()

	node.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	args := net.KnownRequest{
		From: "peer1",
	}
	expectedResp := net.KnownResponse{
		Known: node.core.Known(),
	}

	var out net.KnownResponse
	if err := peer1Trans.RequestKnown(peers[0].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if !reflect.DeepEqual(expectedResp, out) {
		t.Fatalf("KnownResponse should be %#v, not %#v", expectedResp, out)
	}

	node.Shutdown()
}

func TestProcessSync(t *testing.T) {
	keys, peers := initPeers()
	testLogger := common.NewTestLogger(t)

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans, proxy.NewInmemProxy(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans, proxy.NewInmemProxy(testLogger))
	node1.Init()

	head, unknown := node1.core.Diff(node0.core.Known())

	args := net.SyncRequest{
		Head:   head,
		Events: unknown,
	}
	expectedResp := net.SyncResponse{
		Success: true,
	}

	var out net.SyncResponse
	if err := peer1Trans.Sync(peers[0].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if !reflect.DeepEqual(expectedResp, out) {
		t.Fatalf("SyncResponse should be %#v, not %#v", expectedResp, out)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestAddTransaction(t *testing.T) {
	keys, peers := initPeers()
	testLogger := common.NewTestLogger(t)
	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()
	peer0Proxy := proxy.NewInmemProxy(testLogger)

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans, peer0Proxy)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()
	peer1Proxy := proxy.NewInmemProxy(testLogger)

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans, peer1Proxy)
	node1.Init()

	message := "Hello World!"
	peer0Proxy.SubmitTx([]byte(message))

	head, unknown := node1.core.Diff(node0.core.Known())

	args := net.SyncRequest{
		Head:   head,
		Events: unknown,
	}

	var out net.SyncResponse
	if err := peer1Trans.Sync(peers[0].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

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

func initNodes(logger *logrus.Logger) ([]*ecdsa.PrivateKey, []Node) {
	conf := NewConfig(5*time.Millisecond, logger)

	keys, peers := initPeers()
	nodes := []Node{}
	proxies := []*proxy.InmemProxy{}
	for i := 0; i < len(peers); i++ {
		trans, err := net.NewTCPTransport(peers[i].NetAddr,
			nil, 2, time.Second, logger)
		if err != nil {
			logger.Printf(err.Error())
		}
		prox := proxy.NewInmemProxy(logger)
		node := NewNode(conf, keys[i], peers, trans, prox)
		node.Init()
		nodes = append(nodes, node)
		proxies = append(proxies, prox)
	}
	return keys, nodes
}

func runNodes(nodes []Node, gossip bool) {
	for _, n := range nodes {
		go func(node Node) {
			node.Run(gossip)
		}(n)
	}
}

func shutdownNodes(nodes []Node) {
	for _, n := range nodes {
		n.Shutdown()
		n.trans.Close()
	}
}

func getCommittedTransactions(n Node) ([][]byte, error) {
	inmemProxy, ok := n.proxy.(*proxy.InmemProxy)
	if !ok {
		return nil, fmt.Errorf("Error casting to InmemProp")
	}
	res := inmemProxy.GetCommittedTransactions()
	return res, nil
}

/*
h0  |   h2
| \ | / |
|   h1  |
|  /|   |
g02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  g21
|   | / |
|  g10  |
| / |   |
g0  |   g2
| \ | / |
|   g1  |
|  /|   |
f02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  f21
|   | / |
|  f10  |
| / |   |
f0  |   f2
| \ | / |
|   f1  |
|  /|   |
e02 |   |
| \ |   |
|   \   |
|   | \ |
|   |  e21
|   | / |
|  e10  |
| / |   |
e0  e1  e2
0   1    2
*/
func TestTransactionOrdering(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(logger)
	runNodes(nodes, false)

	playbook := []play{
		play{from: 0, to: 1, payload: [][]byte{[]byte("e10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("e21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("e02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("f1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("f0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f2")}},

		play{from: 0, to: 1, payload: [][]byte{[]byte("f10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("f02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("g1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("g0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g2")}},

		play{from: 0, to: 1, payload: [][]byte{[]byte("g10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("g02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("h1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("h0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("h2")}},
	}

	for _, play := range playbook {
		if err := synchronizeNodes(nodes[play.from], nodes[play.to], play.payload); err != nil {
			t.Fatal(err)
		}
	}
	shutdownNodes(nodes)

	expectedConsTransactions := [][]byte{
		[]byte("e10"),
		[]byte("e21"),
		[]byte("e02"),
	}
	for i, n := range nodes {
		consTransactions, err := getCommittedTransactions(n)
		if err != nil {
			t.Fatal(err)
		}
		for j, et := range expectedConsTransactions {
			if at := string(consTransactions[j]); at != string(et) {
				t.Fatalf("node[%d].ConsensusTransactions[%d] should be %s, not %s", i, j, string(et), at)
			}
		}
	}
}

func TestStats(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(logger)
	runNodes(nodes, false)

	playbook := []play{
		play{from: 0, to: 1, payload: [][]byte{[]byte("e10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("e21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("e02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("f1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("f0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f2")}},

		play{from: 0, to: 1, payload: [][]byte{[]byte("f10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("f21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("f02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("g1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("g0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g2")}},

		play{from: 0, to: 1, payload: [][]byte{[]byte("g10")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("g21")}},
		play{from: 2, to: 0, payload: [][]byte{[]byte("g02")}},
		play{from: 0, to: 1, payload: [][]byte{[]byte("h1")}},
		play{from: 1, to: 0, payload: [][]byte{[]byte("h0")}},
		play{from: 1, to: 2, payload: [][]byte{[]byte("h2")}},
	}

	for _, play := range playbook {
		if err := synchronizeNodes(nodes[play.from], nodes[play.to], play.payload); err != nil {
			t.Fatal(err)
		}
	}
	shutdownNodes(nodes)

	stats := nodes[0].GetStats()
	expectedStats := map[string]string{
		"last_consensus_round":   "1",
		"consensus_events":       "6",
		"consensus_transactions": "3",
		"undetermined_events":    "14",
		"transaction_pool":       "0",
		"num_peers":              "2",
	}

	t.Logf("%#v", stats)

	if !reflect.DeepEqual(stats, expectedStats) {
		t.Fatalf("Stats should be %#v, not %#v", expectedStats, stats)
	}
}

func synchronizeNodes(from Node, to Node, payload [][]byte) error {
	toProxy, ok := to.proxy.(*proxy.InmemProxy)
	if !ok {
		return fmt.Errorf("Error casting to InmemProxy")
	}
	for _, t := range payload {
		toProxy.SubmitTx(t)
	}
	known, err := from.requestKnown(to.localAddr)
	if err != nil {
		return err
	}

	head, diff := from.core.Diff(known)
	if err := from.requestSync(to.localAddr, head, diff); err != nil {
		return err
	}

	return nil
}

func TestGossip(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(logger)

	runNodes(nodes, true)
	quit := make(chan int)
	makeRandomTransactions(nodes, quit)

	//wait until all nodes have at least 50 consensus events
	for {
		time.Sleep(10 * time.Millisecond)
		done := true
		for _, n := range nodes {
			if len(n.GetConsensusEvents()) < 50 {
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

	consEvents := [][]string{}
	consTransactions := [][][]byte{}
	for _, n := range nodes {
		consEvents = append(consEvents, n.GetConsensusEvents())
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

func submitTransaction(n Node, tx []byte) error {
	prox, ok := n.proxy.(*proxy.InmemProxy)
	if !ok {
		return fmt.Errorf("Error casting to InmemProp")
	}
	prox.SubmitTx([]byte(tx))
	return nil
}

func makeRandomTransactions(nodes []Node, quit chan int) {
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
