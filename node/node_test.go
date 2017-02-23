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

	node := NewNode(TestConfig(t), keys[0], peers, peer0Trans)
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

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans)
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

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(TestConfig(t), keys[0], peers, peer0Trans)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(TestConfig(t), keys[1], peers, peer1Trans)
	node1.Init()

	message := "Hello World!"
	node0.AddTransaction([]byte(message))

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
	for i := 0; i < len(peers); i++ {
		trans, err := net.NewTCPTransport(peers[i].NetAddr,
			nil, 2, time.Second, logger)
		if err != nil {
			logger.Printf(err.Error())
		}
		node := NewNode(conf, keys[i], peers, trans)
		node.Init()
		nodes = append(nodes, node)
	}
	return nil, nodes
}

func runNodes(nodes []Node) {
	for _, n := range nodes {
		go func(node Node) {
			node.Run(true)
		}(n)
	}
}

func shutdownNodes(nodes []Node) {
	for _, n := range nodes {
		n.Shutdown()
		n.trans.Close()
	}
}

func TestGossip(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(logger)

	runNodes(nodes)

	//wait until all nodes have 5 consensus events
	for {
		time.Sleep(1 * time.Second)
		done := true
		for _, n := range nodes {
			if len(n.GetConsensusEvents()) < 5 {
				done = false
				break
			}
		}
		if done {
			break
		}
	}

	shutdownNodes(nodes)

	for i, e := range nodes[0].GetConsensusEvents()[0:5] {
		for j, n := range nodes[1:len(nodes)] {
			if n.GetConsensusEvents()[i] != e {
				t.Fatalf("nodes[%d].Consensus[%d] and nodes[0].Consensus[%d] are not equal", j, i, i)
			}
		}
	}
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
				node.AddTransaction([]byte(fmt.Sprintf("node%d transaction %d", n, seq[n])))
				seq[n] = seq[n] + 1
				time.Sleep(3 * time.Millisecond)
			}
		}
	}()
}

func TestTransactionOrdering(t *testing.T) {
	logger := common.NewTestLogger(t)
	_, nodes := initNodes(logger)

	runNodes(nodes)
	quit := make(chan int)
	makeRandomTransactions(nodes, quit)
	//wait until all nodes have 5 consensus events
	for {
		time.Sleep(1 * time.Second)
		done := true
		for _, n := range nodes {
			if len(n.GetConsensusEvents()) < 5 {
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

	consTransactions := [][][]byte{}
	for _, n := range nodes {
		nodeTxs, err := n.GetConsensusTransactions()
		if err != nil {
			t.Fatal(err)
		}
		consTransactions = append(consTransactions, nodeTxs)
	}

	min := len(consTransactions[0])
	for k := 1; k < len(consTransactions); k++ {
		if len(consTransactions[k]) < min {
			min = len(consTransactions[k])
		}
	}

	t.Logf("min consensus transactions: %d", min)

	for i, tx := range consTransactions[0][:min] {
		for k, _ := range nodes[1:len(nodes)] {
			if ot := string(consTransactions[k][i]); ot != string(tx) {
				t.Fatalf("nodes[%d].Tx[%d] should be %s not %s", k, i, string(tx), ot)
			}
		}
	}
}
