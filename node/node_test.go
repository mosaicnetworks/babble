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

	"github.com/arrivets/go-swirlds/common"
	"github.com/arrivets/go-swirlds/crypto"
	"github.com/arrivets/go-swirlds/net"
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

func TestListen_Known(t *testing.T) {
	keys, peers := initPeers()

	peer0Trans, err := net.NewTCPTransportWithLogger(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node := NewNode(keys[0], peers, peer0Trans)
	node.Init()

	node.StartAsync()

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
}

func TestListen_Sync(t *testing.T) {
	keys, peers := initPeers()

	peer0Trans, err := net.NewTCPTransportWithLogger(peers[0].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	node0 := NewNode(keys[0], peers, peer0Trans)
	node0.Init()

	node0.StartAsync()

	peer1Trans, err := net.NewTCPTransportWithLogger(peers[1].NetAddr, nil, 2, time.Second, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(keys[1], peers, peer1Trans)
	node1.Init()

	head, unknown := node1.core.Diff(node0.core.Known())
	fmt.Printf("\n unknown[0]: %#v\n", unknown[0])

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
}
