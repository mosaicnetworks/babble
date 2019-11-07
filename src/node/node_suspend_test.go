package node

import (
	"os"
	"testing"
	"time"
)

func TestAutoSuspend(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	// define 5 validators
	keys, peers := initPeers(t, 5)
	genesisPeerSet := clonePeerSet(t, peers.Peers)

	// initialize only three nodes
	nodes := []*Node{
		newNode(peers.Peers[0], keys[0], genesisPeerSet, peers, 1000, 1000, 10, false, "badger", 10*time.Millisecond, t),
		newNode(peers.Peers[1], keys[1], genesisPeerSet, peers, 1000, 1000, 10, false, "badger", 10*time.Millisecond, t),
		newNode(peers.Peers[2], keys[2], genesisPeerSet, peers, 1000, 1000, 10, false, "badger", 10*time.Millisecond, t),
	}
	defer shutdownNodes(nodes)

	// With only 3/5 of nodes gossipping, the cluster should never reach
	// consensus on any Events. So they will keep producing undetermined events
	// until the SuspendLimit is reached (300 by default). When the limit is
	// reached, all nodes should be suspended, an no more undetermined events
	// should be created. Gossip for 10 seconds which should be enough time for
	// the limit to be reached; a non-nil error should be returned because no
	// blocks are ever committed.
	err := gossip(nodes, 1, true, 10*time.Second)
	if err == nil {
		t.Fatal("suspended nodes should not have created blocks")
	}

	if s := nodes[0].getState(); s != Suspended {
		t.Fatalf("nodes[0] should be Suspended, not %v", s)
	}
	if s := nodes[1].getState(); s != Suspended {
		t.Fatalf("nodes[1] should be Suspended, not %v", s)
	}

	node0FirstUE := len(nodes[0].core.GetUndeterminedEvents())
	t.Logf("nodes[0].UndeterminedEvents = %d", node0FirstUE)
	node1FirstUE := len(nodes[1].core.GetUndeterminedEvents())
	t.Logf("nodes[1].UndeterminedEvents = %d", node1FirstUE)

	// Now restart the nodes and hope that they gossip some more, until they
	// create 300 more undetermined events
	nodes[0].Shutdown()
	nodes[1].Shutdown()
	nodes[0] = recycleNode(nodes[0], t)
	nodes[1] = recycleNode(nodes[1], t)

	// Gossip again until they create another SuspendLimit undetermined events.
	err = gossip(nodes, 1, true, 10*time.Second)
	if err == nil {
		t.Fatal("suspended nodes should not have created blocks")
	}

	if s := nodes[0].getState(); s != Suspended {
		t.Fatalf("nodes[0] should be Suspended, not %v", s)
	}
	if s := nodes[1].getState(); s != Suspended {
		t.Fatalf("nodes[1] should be Suspended, not %v", s)
	}

	node0SecondUE := len(nodes[0].core.GetUndeterminedEvents())
	t.Logf("nodes[0].UndeterminedEvents = %d", node0SecondUE)
	node1SecondUE := len(nodes[1].core.GetUndeterminedEvents())
	t.Logf("nodes[1].UndeterminedEvents = %d", node1SecondUE)

	if node0SecondUE-node0FirstUE < nodes[0].conf.SuspendLimit {
		t.Fatalf("nodes[0] should have produced some events between suspensions")
	}

	if node1SecondUE-node1FirstUE < nodes[1].conf.SuspendLimit {
		t.Fatalf("nodes[1] should have produced some events between suspensions")
	}
}
