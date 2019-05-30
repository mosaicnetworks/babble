// +build !unit

package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	bkeys "github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestSuccessiveJoinRequestExtra(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(t, 1)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	node0 := newNode(peerSet.Peers[0], keys[0], peerSet, genesisPeerSet, 1000000, 400, 5, true, "inmem", 10*time.Millisecond, logger, t)
	defer node0.Shutdown()
	node0.RunAsync(true)

	nodes := []*Node{node0}
	//defer drawGraphs(nodes, t)

	target := 20
	for i := 1; i <= 3; i++ {
		peerSet := peers.NewPeerSet(node0.GetPeers())

		key, _ := bkeys.GenerateECDSAKey()
		peer := peers.NewPeer(
			bkeys.PublicKeyHex(&key.PublicKey),
			fmt.Sprintf("127.0.0.1:%d", 4240+i),
			fmt.Sprintf("monika%d", i),
		)
		newNode := newNode(peer, key, peerSet, genesisPeerSet, 1000000, 400, 5, true, "inmem", 10*time.Millisecond, logger, t)

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
		checkPeerSets(nodes, t)

		target = target + 40
	}
}

func TestSuccessiveLeaveRequestExtra(t *testing.T) {
	n := 4

	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(t, n)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, 1000000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	target := 0

	f := func() {
		t.Logf("SUCCESSIVE LEAVE n=%d", n)
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

func TestSimultaneousLeaveRequestExtra(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, 1000000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
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
	keys, peerSet := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, 1000000, 1000, 5, true, "inmem", 5*time.Millisecond, logger, t)
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

	key, _ := bkeys.GenerateECDSAKey()
	peer := peers.NewPeer(
		bkeys.PublicKeyHex(&key.PublicKey),
		fmt.Sprint("127.0.0.1:4242"),
		"new node",
	)
	newNode := newNode(peer, key, peerSet, genesisPeerSet, 1000000, 200, 5, true, "inmem", 10*time.Millisecond, logger, t)
	defer newNode.Shutdown()

	// Run parallel routine to check newNode eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second) //TODO this process has been amended - may not be in CatchingUp state
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

//TestAddingAndRemovingPeers is a complex test. The broad brush outline of the process is as follows:
//
//	1 Construct a network of 5 nodes and build a history of transactions.
//	2 Remove a validator from the peer list
//  3 Build more history
//  4 Add a new validator and sync without using fast sync
//		i.e. apply the whole hashgraph
//  5 Build more history
//	6 Add another validator and sync without fast sync
//  7 Add more history and check that all peers have the same state
//  8 Add Node 7, 8
//  9 Remove Nodes 0 to 3
//  10 Add Node 9
//
//	Nodes 0 to 3 are on until step 9
//  Node 4 is removed in step 2
//  Node 5 is added in step 4
//  Node 6 is added in step 6
//  Nodes 7,8 are added in Step 8
//  Node 9 is added in Step 10
func TestJoiningAndLeavingExtra(t *testing.T) {

	logger := common.NewTestLogger(t)

	// Step 1 - Create 5 nodes

	keys, peerlist := initPeers(t, 10)

	logPeerList(t, peerlist, "Log Peers")

	peers01234 := clonePeerSet(t, peerlist.Peers[0:5])
	peers0123 := clonePeerSet(t, peerlist.Peers[0:4])
	peers01235 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:6]...))    // Step 4
	peers012356 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:7]...))   // Step 6
	peers0123567 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:8]...))  // Step 8
	peers01235678 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:9]...)) // Step 8
	//	peers5678 := clonePeerSet(t, peerlist.Peers[5:9])                                  // Step 9
	peers5678 := clonePeerSet(t, peerlist.Peers[5:9]) // Step 10

	genesisPeerSet := clonePeerSet(t, peers01234.Peers)

	logPeerList(t, peers01234, "Peers01234")
	logPeerList(t, peers01235, "Peers01235")
	logPeerList(t, peers012356, "Peers012356")
	logPeerList(t, peers0123567, "Peers0123567")
	logPeerList(t, peers01235678, "Peers01235678")
	logPeerList(t, peers5678, "Peers5678")
	logPeerList(t, genesisPeerSet, "genesisPeerSet")

	t.Log("Step 1")
	// 5 nodes are live

	logPeerList(t, peerlist, "Log Peers")

	nodes01234 := initNodes(keys[0:5], peers01234, genesisPeerSet, 100000, 400, 15, false, "inmem", 10*time.Millisecond, logger, t) //make cache high to draw graphs
	//	defer shutdownNodesSlice(t, nodes01234, []uint{0, 1, 2, 3})
	//defer drawGraphs(nodes, t)

	// Step 1b - gossip and build history
	t.Log("Step 1b")

	target := nodes01234[0].core.hg.Store.LastBlockIndex() + 1

	err := gossip(nodes01234, target+20, false, 10*time.Second)
	if err != nil {
		t.Fatal("Step 1b gossip", err)
	}
	checkGossip(nodes01234, target, t)

	checkPeerSets(nodes01234, t)

	// Step 2 - Node 4 leaves
	t.Log("Step 2")

	node4 := nodes01234[4]

	// Pause for realism
	// time.Sleep(2 * time.Second)

	err = node4.Leave()
	if err != nil {
		t.Fatal("Step 2 Leave", err)
	}

	// New nodes array without node 4
	nodes0123 := nodes01234[0:4]

	time.Sleep(400 * time.Millisecond)

	checkPeerSets(nodes0123, t)

	// Step 3 - More history
	t.Log("Step 3")

	target = nodes0123[0].core.hg.Store.LastBlockIndex() + 1
	err = gossip(nodes0123, target+30, false, 10*time.Second)
	if err != nil {
		t.Fatal("Step 3 Gossip", err)
	}
	checkGossip(nodes0123, target, t)

	// Step 4 Add a new validator (node 5) and sync without using fast sync
	t.Log("Step 4")

	logPeerList(t, peerlist, "Log Peers 5")

	node5 := newNode(peerlist.Peers[5], keys[5], peers0123, genesisPeerSet, 1000000, 100, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer node5.Shutdown()

	// New nodes array without node 4
	nodes01235 := append(append([]*Node{}, nodes0123...), node5)

	// Step 5 Build more history
	t.Log("Step 5")
	logNodeList(t, nodes01235, "Nodes 01235")

	node5.RunAsync(true)

	time.Sleep(2 * time.Second)

	target = nodes01235[0].core.hg.Store.LastBlockIndex() + 1

	t.Logf("Target block is %d", target+30)

	err = gossip(nodes01235, target+30, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Step 5b")

	checkGossip(nodes01235, target, t)

	// Step 6 Add another validator (node 6) and sync without fast sync
	t.Log("Step 6")

	logPeerList(t, peerlist, "Log Peers 6")
	node6 := newNode(peerlist.Peers[6], keys[6], peers01235, genesisPeerSet, 1000000, 100, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer node6.Shutdown()

	nodes012356 := append(append([]*Node{}, nodes01235...), node6)
	node6.RunAsync(true)

	// Verify new join has updated the PeerSets

	// We sleep to ensure join process has completed.
	time.Sleep(2 * time.Second)

	for nodeIdx, tmpnode := range nodes012356 {
		ps0, _ := tmpnode.core.hg.Store.GetAllPeerSets()
		t.Logf("Node peer list %d", nodeIdx)
		t.Log("Node peer list ", ps0)
	}

	checkPeerSets(nodes012356, t)
	// Step 7 Add more history and check that all peers have the same state
	t.Log("Step 7")

	target = nodes012356[0].core.hg.Store.LastBlockIndex() + 1
	err = gossip(nodes012356, target+10, false, 5*time.Second)
	if err != nil {
		t.Log("Fatal Error", err)
		t.Fatal(err)
	}

	return //TODO remove this line

	t.Log("Step 7b")
	checkGossip(nodes012356, target, t)

	//  Step 8 Add Node 7, 8
	t.Log("Step 8")

	logPeerList(t, peerlist, "Log Peers 7")
	node7 := newNode(peerlist.Peers[7], keys[7], peers012356, genesisPeerSet, 1000000, 100, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer node7.Shutdown()

	nodes0123567 := append(append([]*Node{}, nodes012356...), node7)
	node7.RunAsync(true)

	t.Log("Step 8b")

	target = nodes0123567[0].core.hg.Store.LastBlockIndex() + 1
	err = gossip(nodes0123567, target+12, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes0123567, target, t)

	logPeerList(t, peerlist, "Log Peers 8")
	node8 := newNode(peerlist.Peers[8], keys[8], peers0123567, genesisPeerSet, 1000000, 100, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer node8.Shutdown()

	nodes01235678 := append(append([]*Node{}, nodes0123567...), node8)
	node8.RunAsync(true)

	// Step 8b Add more history and check that all peers have the same state
	t.Log("Step 8c")

	target = nodes01235678[0].core.hg.Store.LastBlockIndex() + 1
	err = gossip(nodes01235678, target+20, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes01235678, target, t)

	//  Step 9 Remove Nodes 0 to 3

	t.Log("Step 9")
	node3 := nodes0123[3]
	err = node3.Leave()
	if err != nil {
		t.Fatal("Step 9 Leave", err)
	}

	t.Log("Step 9b")
	node2 := nodes0123[2]
	err = node2.Leave()
	if err != nil {
		t.Fatal("Step 9b Leave", err)
	}

	t.Log("Step 9c")
	node1 := nodes0123[1]
	err = node1.Leave()
	if err != nil {
		t.Fatal("Step 9c Leave", err)
	}

	t.Log("Step 9d")
	node0 := nodes0123[0]
	err = node0.Leave()
	if err != nil {
		t.Fatal("Step 9d Leave", err)
	}

	// New nodes array without nodes 0 to 3
	nodes5678 := nodes01235678[4:]

	target += 13
	err = gossip(nodes5678, target, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes5678, 0, t)

	//  Step 10 Add Node 9

	t.Log("Step 10")

	logPeerList(t, peerlist, "Log Peers 9")
	node9 := newNode(peerlist.Peers[9], keys[9], peers5678, genesisPeerSet, 1000000, 100, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer node9.Shutdown()

	nodes56789 := append(append([]*Node{}, nodes5678...), node9)
	node9.RunAsync(true)

	target += 17
	err = gossip(nodes56789, target, false, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes56789, 0, t)

	// Check everything is in sync

	// Last Step Shutdown cleanly
	// Is handled by defer commands set as objects are created

	t.Log("Final Step")

	return

}
