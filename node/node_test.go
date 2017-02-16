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
	"reflect"
	"testing"
	"time"

	"log"

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

	peer0Trans, err := net.NewTCPTransportWithLogger(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node := NewNode(DefaultConfig(), keys[0], peers, peer0Trans)
	node.Init()

	node.RunAsync(false)

	peer1Trans, err := net.NewTCPTransportWithLogger(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
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

	peer0Trans, err := net.NewTCPTransportWithLogger(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(DefaultConfig(), keys[0], peers, peer0Trans)
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransportWithLogger(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(DefaultConfig(), keys[1], peers, peer1Trans)
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

func initNodes(logger *log.Logger) ([]*ecdsa.PrivateKey, []Node) {
	conf := DefaultConfig()
	conf.Logger = logger
	keys, peers := initPeers()
	nodes := []Node{}
	for i := 0; i < len(peers); i++ {
		trans, err := net.NewTCPTransportWithLogger(peers[i].NetAddr,
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
	}
}

func BenchmarkGossip(b *testing.B) {
	logger := common.NewBenchmarkLogger(b)
	_, nodes := initNodes(logger)

	runNodes(nodes)

	//wait until all nodes have 5 consensus events
	for {
		time.Sleep(1 * time.Second)
		done := true
		for _, n := range nodes {
			if len(n.GetConsensus()) < 5 {
				done = false
				break
			}
		}
		if done {
			break
		}
	}

	shutdownNodes(nodes)

	for i, e := range nodes[0].GetConsensus()[0:5] {
		for j, n := range nodes[1:len(nodes)] {
			if n.GetConsensus()[i] != e {
				logger.Printf("nodes[%d].Consensus[%d] and nodes[0].Consensus[%d] are not equal", j, i, i)
			}
		}
	}
}
