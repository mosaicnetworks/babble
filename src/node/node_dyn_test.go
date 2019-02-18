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
	nodes := initNodes(keys, peers, 100000, 1000, "inmem", 5*time.Millisecond, logger, t)
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
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
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
	newNode := newNode(peer, key, peerSet, 1000, 1000, "inmem", 5*time.Millisecond, logger, t)
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

func TestLeaveRequest(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	leavingNode := nodes[3]

	err = leavingNode.leave()
	if err != nil {
		t.Fatal(err)
	}

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes[0:3], secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes[0:3], 0, t)
	checkPeerSets(nodes[0:3], t)
}

func TestJoinFull(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	initialNodes := initNodes(keys, peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
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
	newNode := newNode(peer, key, peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
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

func TestOrganicGrowth(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(1)

	node0 := newNode(peerSet.Peers[0], keys[0], peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
	defer node0.Shutdown()
	node0.RunAsync(true)

	nodes := []*Node{node0}
	//defer drawGraphs(nodes, t)

	target := 20
	for i := 1; i <= 3; i++ {
		peerSet := peers.NewPeerSet(node0.GetPeers())

		key, _ := crypto.GenerateECDSAKey()
		peer := peers.NewPeer(
			fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
			fmt.Sprintf("127.0.0.1:%d", 4240+i),
		)
		newNode := newNode(peer, key, peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)

		logger.Debugf("starting new node %d, %d", i, newNode.ID())
		defer newNode.Shutdown()
		newNode.RunAsync(true)

		nodes = append(nodes, newNode)

		//Gossip some more
		err := bombardAndWait(nodes, target, 10*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		start := newNode.core.hg.FirstConsensusRound
		checkGossip(nodes, *start, t)

		target = target + 40
	}
}

func checkPeerSets(nodes []*Node, t *testing.T) {
	node0FP, err := nodes[0].core.hg.Store.GetAllPeerSets()
	if err != nil {
		t.Fatal(err)
	}
	for i := range nodes[1:] {
		nodeiFP, err := nodes[i].core.hg.Store.GetAllPeerSets()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(node0FP, nodeiFP) {
			t.Logf("Node 0 PeerSets: %v", node0FP)
			t.Logf("Node %d PeerSets: %v", i, nodeiFP)
			t.Fatalf("PeerSets defer")
		}
	}
}
