package node

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestMonologue(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(1)
	nodes := initNodes(keys, peers, 100000, 1000, "inmem", logger, t)
	//defer drawGraphs(nodes, t)

	target := 50
	err := gossip(nodes, target, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes, 0, t)
}

func TestJoinRequest(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", logger, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	key, _ := crypto.GenerateECDSAKey()
	peer := peers.NewPeer(
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
		fmt.Sprint("127.0.0.1:4242"),
	)
	newNode := newNode(peer, key, peerSet, 1000, 1000, "inmem", logger, t)
	defer newNode.Shutdown()

	err = newNode.join()
	if err != nil {
		t.Fatal(err)
	}

	//Gossip some more
	secondTarget := target + 30
	err = bombardAndWait(nodes, secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)
	checkPeerSets(nodes, t)
}

func TestJoinFull(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	initialNodes := initNodes(keys, peerSet, 1000000, 400, "inmem", logger, t)
	defer shutdownNodes(initialNodes)

	target := 30
	err := gossip(initialNodes, target, false, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(initialNodes, 0, t)

	key, _ := crypto.GenerateECDSAKey()
	peer := peers.NewPeer(
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
		fmt.Sprint("127.0.0.1:4242"),
	)
	newNode := newNode(peer, key, peerSet, 1000000, 400, "inmem", logger, t)
	defer newNode.Shutdown()

	//Run parallel routine to check newNode eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for newNode to enter CatchingUp state")
			default:
			}
			if newNode.getState() == CatchingUp {
				break
			}
		}
	}()

	newNode.RunAsync(true)

	nodes := append(initialNodes, newNode)

	//defer drawGraphs(nodes, t)

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes, secondTarget, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := newNode.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
	checkPeerSets(nodes, t)
}

func TestOneTwo(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(1)

	node0 := newNode(peerSet.Peers[0], keys[0], peerSet, 1000000, 400, "inmem", logger, t)
	node0.conf.HeartbeatTimeout = 100 * time.Millisecond
	defer node0.Shutdown()
	node0.RunAsync(true)

	key, _ := crypto.GenerateECDSAKey()
	peer := peers.NewPeer(
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
		fmt.Sprint("127.0.0.1:4242"),
	)
	newNode := newNode(peer, key, peerSet, 1000000, 400, "inmem", logger, t)
	newNode.conf.HeartbeatTimeout = 100 * time.Millisecond
	defer newNode.Shutdown()
	newNode.RunAsync(true)

	nodes := []*Node{node0, newNode}

	defer drawGraphs(nodes, t)

	//Gossip some more
	target := 20
	err := bombardAndWait(nodes, target, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := newNode.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
	checkPeerSets(nodes, t)
}

// func TestPullAfterJoin(t *testing.T) {
// 	logger := common.NewTestLogger(t)

// 	keys, peerSet := initPeers(3)
// 	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", logger, t)

// 	defer shutdownNodes(nodes)
// 	defer drawGraphs(nodes, t)

// 	target := 50
// 	err := gossip(nodes, target, false, 3*time.Second)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	checkGossip(nodes, 0, t)

// 	key, _ := crypto.GenerateECDSAKey()
// 	peer := peers.NewPeer(
// 		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
// 		fmt.Sprint("127.0.0.1:4242"),
// 	)
// 	newNode := newNode(peer, key, peerSet, 1000, 1000, "inmem", logger, t)

// 	err = newNode.join()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = newNode.fastForward()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	frameRound := newNode.core.hg.FirstConsensusRound

// 	frame, err := newNode.core.hg.Store.GetFrame(*frameRound)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	badRounds := false
// 	for _, ev := range frame.Events {
// 		realEv, err := nodes[0].core.hg.Store.GetEvent(ev.Hex())
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		if *realEv.GetRound() != *ev.GetRound() {
// 			t.Logf("Event %s round should be %d, not %d", ev.Hex(), *realEv.GetRound(), *ev.GetRound())
// 			badRounds = true
// 		}
// 	}

// 	if badRounds {
// 		t.Fatalf("Bad Rounds")
// 	}
// }

// func TestPeerLeaveRequest(t *testing.T) {
// 	logger := common.NewTestLogger(t)

// 	keys, peerSet := initPeers(4)
// 	nodes := initNodes(keys, peerSet, 1000, 1000, "inmem", logger, t)

// 	runNodes(nodes, true)

// 	target := 50

// 	err := bombardAndWait(nodes, target, 3*time.Second)
// 	if err != nil {
// 		t.Fatal("Error bombarding: ", err)
// 	}

// 	nodes[1].Shutdown()
// 	nodes = append([]*Node{nodes[0]}, nodes[2:]...)

// 	target = 50

// 	err = bombardAndWait(nodes, target, 3*time.Second)
// 	if err != nil {
// 		t.Fatal("Error bombarding: ", err)
// 	}

// 	for i := range nodes {
// 		if nodes[i].core.peers.Len() != 3 {
// 			t.Errorf("Node %d should have %d peers, not %d", i, 3, nodes[i].core.peers.Len())
// 		}
// 	}
// }

func checkPeerSets(nodes []*Node, t *testing.T) {
	node0FP, err := nodes[0].core.hg.Store.GetFuturePeerSets(-1)
	if err != nil {
		t.Fatal(err)
	}
	for i := range nodes[1:] {
		nodeiFP, err := nodes[i].core.hg.Store.GetFuturePeerSets(-1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(node0FP, nodeiFP) {
			t.Logf("Node 0 FuturePeerSets: %v", node0FP)
			t.Logf("Node %d FuturePeerSets: %v", i, nodeiFP)
			t.Fatalf("FuturePeerSets defer")
		}
	}
}
