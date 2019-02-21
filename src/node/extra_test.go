// +build !unit

package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestResetExtra(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(4)
	nodes := initNodes(keys, peers, 100000, 100, "inmem", 10*time.Millisecond, logger, t) //make cache high to draw graphs
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 50
	err := gossip(nodes, target, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	node0 := nodes[0]
	node0.Shutdown()

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
	secondTarget := target + 50
	err = bombardAndWait(nodes, secondTarget, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := node0.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestSuccessiveJoinRequestExtra(t *testing.T) {
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
			"monika",
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

func TestSuccessiveLeaveRequestExtra(t *testing.T) {
	n := 4

	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(n)
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	target := 0

	f := func() {
		//defer drawGraphs(nodes, t)
		target += 30
		err := gossip(nodes, target, false, 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		checkGossip(nodes, 0, t)

		leavingNode := nodes[n-1]

		err = leavingNode.Leave()
		if err != nil {
			t.Fatal(err)
		}

		if n == 1 {
			return
		}

		nodes = nodes[0 : n-1]

		//Gossip some more
		target += 50
		err = bombardAndWait(nodes, target, 6*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		checkGossip(nodes, 0, t)
		checkPeerSets(nodes, t)
	}

	for n > 0 {
		f()
		n--
	}
}

func TestSimultaneusLeaveRequestExtra(t *testing.T) {
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
	leavingNode2 := nodes[2]

	err = leavingNode.Leave()
	if err != nil {
		t.Fatal(err)
	}

	err = leavingNode2.Leave()
	if err != nil {
		t.Fatal(err)
	}

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes[0:2], secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes[0:2], 0, t)
	checkPeerSets(nodes[0:2], t)
}

func TestJoinLeaveRequestExtra(t *testing.T) {
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

	err = leavingNode.Leave()
	if err != nil {
		t.Fatal(err)
	}

	key, _ := crypto.GenerateECDSAKey()
	peer := peers.NewPeer(
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
		fmt.Sprint("127.0.0.1:4242"),
		"new node",
	)
	newNode := newNode(peer, key, peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
	defer newNode.Shutdown()

	// Run parallel routine to check newNode eventually reaches CatchingUp state.
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

	// replace leaving node with new node
	nodes[3] = newNode

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes, secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := newNode.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
	checkPeerSets(nodes, t)
}
