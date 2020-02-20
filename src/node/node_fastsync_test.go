package node

import (
	"reflect"
	"testing"
	"time"

	_state "github.com/mosaicnetworks/babble/src/node/state"
)

/*

Test FastForward and state transitions when fast-sync is enabled.

*/

func TestFastForward(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	nodes := initNodes(keys, peers, genesisPeerSet, 1000, 1000, 5, false, "inmem", 5*time.Millisecond, t)
	defer shutdownNodes(nodes)

	target := 20
	err := gossip(nodes[1:], target, false)
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

	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	// Initialize 4 nodes. The last one has fast-sync enabled. We will run the
	// first 3 nodes first, and try to get the 4th one to catch-up.
	nodes := []*Node{
		newNode(peers.Peers[0], keys[0], genesisPeerSet, peers, 10000, 1000, 10, false, "inmem", 10*time.Millisecond, t),
		newNode(peers.Peers[1], keys[1], genesisPeerSet, peers, 10000, 1000, 10, false, "inmem", 10*time.Millisecond, t),
		newNode(peers.Peers[2], keys[2], genesisPeerSet, peers, 10000, 1000, 10, false, "inmem", 10*time.Millisecond, t),
		newNode(peers.Peers[3], keys[3], genesisPeerSet, peers, 10000, 1000, 10, true, "inmem", 10*time.Millisecond, t),
	}
	defer shutdownNodes(nodes)
	//defer drawGraphs(normalNodes, t)

	// create 10 blocks with 3/4 nodes
	target := 10
	err := gossip(nodes[:3], target, false)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(nodes[:3], 0, t)

	// Run parallel routine to check nodes[3] reaches CatchingUp state within 6
	// seconds.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Fatal Timeout waiting for node0 to enter CatchingUp state")
			default:
			}
			if nodes[3].GetState() == _state.CatchingUp {
				break
			}
		}
	}()

	nodes[3].RunAsync(true)

	// Gossip some more with all nodes
	newTarget := target + 20
	err = bombardAndWait(nodes, newTarget)
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}

	// check blocks starting at where nodes[3] joined
	start := nodes[3].core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}

func TestFastSync(t *testing.T) {
	keys, peers := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peers.Peers)

	// all nodes have fast-sync enabled
	// make cache high to draw graphs
	nodes := initNodes(keys, peers, genesisPeerSet, 100000, 400, 5, true, "inmem", 10*time.Millisecond, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	node0 := nodes[0]
	node0.Shutdown()

	secondTarget := target + 30
	err = bombardAndWait(nodes[1:], secondTarget)
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}
	checkGossip(nodes[1:], 0, t)

	//Can't re-run it; have to reinstantiate a new node.
	node0 = recycleNode(node0, t)
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
			if node0.GetState() == _state.CatchingUp {
				break
			}
		}
	}()

	node0.RunAsync(true)

	//Gossip some more
	thirdTarget := secondTarget + 50
	err = bombardAndWait(nodes, thirdTarget)
	if err != nil {
		t.Error("Fatal Error 3", err)
		t.Fatal(err)
	}

	start := node0.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
}
