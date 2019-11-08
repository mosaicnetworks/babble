package node

import (
	"os"
	"testing"
	"time"
)

func TestAutoSuspend(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	// define 3 validators
	keys, peers := initPeers(t, 3)
	genesisPeerSet := clonePeerSet(t, peers.Peers)

	// initialize only 2 nodes
	nodes := []*Node{
		newNode(peers.Peers[0], keys[0], genesisPeerSet, peers, 1000, 1000, 10, false, "badger", 10*time.Millisecond, t),
		newNode(peers.Peers[1], keys[1], genesisPeerSet, peers, 1000, 1000, 10, false, "badger", 10*time.Millisecond, t),
	}
	defer shutdownNodes(nodes)

	// With only 2/3 of nodes gossipping, the cluster should never reach
	// consensus on any Events. So they will keep producing undetermined events
	// until the SuspendLimit is reached (300 by default). When the limit is
	// reached, all nodes should be suspended, an no more undetermined events
	// should be created. Gossip for 10 seconds which should be enough time for
	// the limit to be reached; a non-nil error should be returned because no
	// blocks are ever committed.
	// err := gossip(nodes, 1, true, 10*time.Second)
	// if err == nil {
	// 	t.Fatal("suspended nodes should not have created blocks")
	// }
	// Run parallel routine to check nodes[3] reaches CatchingUp state within 6
	// seconds.
	runNodes(nodes, true)
	submitTransaction(nodes[0], []byte("the tx that will never be committed"))
	waitSuspend(nodes, 10*time.Second, t)

	if s := nodes[0].getState(); s != Suspended {
		t.Fatalf("nodes[0] should be Suspended, not %v. UndeterminedEvents: %d",
			s, len(nodes[0].core.GetUndeterminedEvents()))
	}
	if s := nodes[1].getState(); s != Suspended {
		t.Fatalf("nodes[1] should be Suspended, not %v. UndeterminedEvents: %d",
			s, len(nodes[1].core.GetUndeterminedEvents()))
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
	runNodes(nodes, true)
	submitTransaction(nodes[0], []byte("the tx that will never be committed"))
	waitSuspend(nodes, 10*time.Second, t)

	if s := nodes[0].getState(); s != Suspended {
		t.Fatalf("nodes[0] should be Suspended, not %v. UndeterminedEvents: %d",
			s, len(nodes[0].core.GetUndeterminedEvents()))
	}
	if s := nodes[1].getState(); s != Suspended {
		t.Fatalf("nodes[1] should be Suspended, not %v. UndeterminedEvents: %d",
			s, len(nodes[1].core.GetUndeterminedEvents()))
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

func waitSuspend(nodes []*Node, timeout time.Duration, t *testing.T) {
	stopper := time.After(timeout)
	for {
		select {
		case <-stopper:
			t.Fatalf("TIMEOUT waiting for nodes to be suspended")
		default:
		}
		time.Sleep(10 * time.Millisecond)
		done := true
		for _, n := range nodes {
			if n.getState() != Suspended {
				done = false
				break
			}
		}
		if done {
			break
		}
	}
}
