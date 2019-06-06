package node

import (
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
)

/*

Test FastForward and state transitions when fast-sync is enabled.

*/

func TestFastForward(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, false, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	target := 20
	err := gossip(nodes[1:], target, false, 6*time.Second)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}

	err = nodes[0].fastForward()
	if err != nil {
		t.Fatalf("Fatal Error FastForwarding: %s", err)
	}

	lbi := nodes[0].core.GetLastBlockIndex()
	if lbi <= 0 {
		t.Fatalf("Fatal LastBlockIndex is too low: %d", lbi)
	}

	sBlock, err := nodes[0].GetBlock(lbi)
	if err != nil {
		t.Fatalf("Fatal Error retrieving latest Block from reset hashgraph: %v", err)
	}

	expectedBlock, err := nodes[1].GetBlock(lbi)
	if err != nil {
		t.Fatalf("Fatal Failed to retrieve block %d from node1: %v", lbi, err)
	}

	if !reflect.DeepEqual(sBlock.Body, expectedBlock.Body) {
		t.Fatalf("Fatal Blocks defer")
	}
}

func TestCatchUp(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	// We don't initialize the first node because, since it will stay passive
	// during the first part of the test, too many requests would be queued in
	// its TCP socket (even if it is not running). This is because the socket is
	// setup to listen regardless of whether a node is running or not. We should
	// probably change this at some point.

	normalNodes := initNodes(keys[1:], peers, genesisPeerSet, 1000000, 100, 5, false, "inmem", 50*time.Millisecond, logger, t)
	defer shutdownNodes(normalNodes)
	//defer drawGraphs(normalNodes, t)

	target := 10
	err := gossip(normalNodes, target, false, 6*time.Second)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(normalNodes, 0, t)

	//node0 has fast-sync enabled
	node0 := newNode(peers.Peers[0], keys[0], peers, genesisPeerSet, 1000000, 100, 5, true, "inmem", 10*time.Millisecond, logger, t)
	defer node0.Shutdown()

	//Run parallel routine to check node0 eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Fatal Timeout waiting for node0 to enter CatchingUp state")
			default:
			}
			if node0.getState() == CatchingUp {
				break
			}
		}
	}()

	node0.RunAsync(true)

	nodes := append(normalNodes, node0)

	//Gossip some more with all nodes
	newTarget := target + 20
	err = bombardAndWait(nodes, newTarget, 10*time.Second)
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}

	start := node0.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestFastSync(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	//all nodes have fast-sync enabled
	nodes := initNodes(keys, peers, genesisPeerSet, 100000, 400, 5, true, "inmem", 10*time.Millisecond, logger, t) //make cache high to draw graphs
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 10*time.Second)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	node0 := nodes[0]
	node0.Shutdown()

	secondTarget := target + 30
	err = bombardAndWait(nodes[1:], secondTarget, 10*time.Second)
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}
	checkGossip(nodes[1:], 0, t)

	//Can't re-run it; have to reinstantiate a new node.
	node0 = recycleNode(node0, logger, t)
	nodes[0] = node0

	//Run parallel routine to check node0 eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Fatal Timeout waiting for node0 to enter CatchingUp state")
			default:
			}
			if node0.getState() == CatchingUp {
				break
			}
		}
	}()

	node0.RunAsync(true)

	//Gossip some more
	thirdTarget := secondTarget + 50
	err = bombardAndWait(nodes, thirdTarget, 10*time.Second)
	if err != nil {
		t.Error("Fatal Error 3", err)
		t.Fatal(err)
	}

	start := node0.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}
