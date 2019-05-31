// +build !unit

package node

import (
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	bkeys "github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

func TestSuccessiveJoinRequestExtra(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(t, 1)
	genesisPeerSet := clonePeerSet(t, peerSet.Peers)
	validators := clonePeerSet(t, peerSet.Peers)

	node0 := newNode(peerSet.Peers[0], keys[0], peerSet, genesisPeerSet, validators, 1000000, 400, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer node0.Shutdown()
	node0.RunAsync(true)

	nodes := []*Node{node0}

	target := 10
	for i := 1; i <= 3; i++ {
		peerSet := peers.NewPeerSet(node0.GetPeers())
		validators := clonePeerSet(t, peerSet.Peers)

		key, _ := bkeys.GenerateECDSAKey()
		peer := peers.NewPeer(
			bkeys.PublicKeyHex(&key.PublicKey),
			fmt.Sprintf("127.0.0.1:%d", 4240+i),
			fmt.Sprintf("monika%d", i),
		)
		newNode := newNode(peer, key, peerSet, genesisPeerSet, validators, 1000000, 400, 5, false, "inmem", 10*time.Millisecond, logger, t)

		logger.Debugf("starting new node %d, %d", i, newNode.ID())
		defer newNode.Shutdown()
		newNode.RunAsync(true)

		nodes = append(nodes, newNode)

		//Gossip some more
		err := bombardAndWait(nodes, target, 10*time.Second)
		if err != nil {
			t.Error("Fatal Error in TestSuccessiveJoinRequestExtra", err)
			t.Fatal(err)
		}

		start := newNode.core.hg.FirstConsensusRound
		checkGossip(nodes, *start, t)
		checkPeerSets(nodes, t)

		target = target + 10
	}

	// Pause before exiting
	time.Sleep(2 * time.Second)
}

func TestSuccessiveLeaveRequestExtra(t *testing.T) {
	n := 4

	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(t, n)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)
	validators := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, validators, 1000000, 1000, 20, false, "inmem", 10*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	target := 0

	f := func() {
		t.Logf("SUCCESSIVE LEAVE n=%d", n)
		//defer drawGraphs(nodes, t)
		target += 30
		err := gossip(nodes, target, false, 4*time.Second)
		if err != nil {
			t.Error("Fatal Error", err)
			t.Fatal(err)
		}
		checkGossip(nodes, 0, t)

		leavingNode := nodes[n-1]

		err = leavingNode.Leave()
		if err != nil {
			t.Error("Fatal Error 2", err)
			t.Fatal(err)
		}

		if n == 1 {
			return
		}

		nodes = nodes[0 : n-1]

		//Gossip some more
		target += 50
		err = bombardAndWait(nodes, target, 8*time.Second)
		if err != nil {
			t.Error("Fatal Error 3", err)
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
	validators := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, validators, 1000000, 1000, 5, false, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Error("Fatal Error 1", err)
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	leavingNode := nodes[3]
	leavingNode2 := nodes[2]

	err = leavingNode.Leave()
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}

	err = leavingNode2.Leave()
	if err != nil {
		t.Error("Fatal Error 3", err)
		t.Fatal(err)
	}

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes[0:2], secondTarget, 6*time.Second)
	if err != nil {
		t.Error("Fatal Error 4", err)
		t.Fatal(err)
	}
	checkGossip(nodes[0:2], 0, t)
	checkPeerSets(nodes[0:2], t)
}

func TestJoinLeaveRequestExtra(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)
	validators := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, validators, 1000000, 1000, 5, false, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	leavingNode := nodes[3]

	err = leavingNode.Leave()
	if err != nil {
		t.Error("Fatal Error 2", err)
		t.Fatal(err)
	}

	key, _ := bkeys.GenerateECDSAKey()
	peer := peers.NewPeer(
		bkeys.PublicKeyHex(&key.PublicKey),
		fmt.Sprint("127.0.0.1:4242"),
		"new node",
	)
	newNode := newNode(peer, key, peerSet, genesisPeerSet, validators, 1000000, 200, 5, false, "inmem", 10*time.Millisecond, logger, t)
	defer newNode.Shutdown()

	// Run parallel routine to check newNode eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second) //TODO this process has been amended - may not be in CatchingUp state
	go func() {
		for {
			select {
			case <-timeout:

				t.Error("Fatal Error - Timeout waiting for newNode to enter CatchingUp state")
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
		t.Error("Fatal Error 3", err)
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

	keys, peerlist := initPeers(t, 10)
	// Prebuild Peers
	peers01234 := clonePeerSet(t, peerlist.Peers[0:5])
	peers0123 := clonePeerSet(t, peerlist.Peers[0:4])
	peers01235 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:6]...))   // Step 4
	peers012356 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:7]...))  // Step 6
	peers0123567 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:8]...)) // Step 8
	peers5678 := clonePeerSet(t, peerlist.Peers[5:9])                                                                // Step 10
	genesisPeerSet := clonePeerSet(t, peers01234.Peers)                                                              // Step 10
	validators := clonePeerSet(t, peers01234.Peers)

	t.Log("Step 1")
	nodes01234 := initNodes(keys[0:5], peers01234, genesisPeerSet, validators, 100000, 400, 15, false, "inmem", 10*time.Millisecond, logger, t) //make cache high to draw graphs

	// Step 1b - gossip and build history
	gossipAndCheck(nodes01234, 20, 8, "Step 1b", false, t)
	checkPeerSets(nodes01234, t)

	// Step 2 - Node 4 leaves
	leaveNode("Step 2", nodes01234[4], t)

	// New nodes array without node 4
	nodes0123 := nodes01234[0:4]
	time.Sleep(400 * time.Millisecond)
	checkPeerSets(nodes0123, t)

	// Step 3 - More history
	gossipAndCheck(nodes0123, 30, 8, "Step 3", false, t)

	// Step 4 Add a new validator (node 5) and sync without using fast sync
	node5, nodes01235 := launchNodeAndGossip("Step 4 and 5", nodes0123, peerlist.Peers[5], keys[5],
		peers0123, genesisPeerSet, validators, 1000000, 100, 10, false,
		"inmem", 10*time.Millisecond, logger, t, 20, 8, true)
	defer node5.Shutdown()

	// Step 6 Add another validator (node 6) and sync without fast sync
	// Step 7 Add more history and check that all peers have the same state

	node6, nodes012356 := launchNodeAndGossip("Step 6 and 7", nodes01235, peerlist.Peers[6], keys[6],
		peers01235, genesisPeerSet, validators, 1000000, 100, 10, false,
		"inmem", 10*time.Millisecond, logger, t, 15, 8, true)
	defer node6.Shutdown()

	//  Step 8 Add Node 7, 8

	node7, nodes0123567 := launchNodeAndGossip("Step 8", nodes012356, peerlist.Peers[7], keys[7],
		peers012356, genesisPeerSet, validators, 1000000, 100, 10, false,
		"inmem", 10*time.Millisecond, logger, t, 14, 8, false)
	defer node7.Shutdown()

	node8, nodes01235678 := launchNodeAndGossip("Step 8b", nodes0123567, peerlist.Peers[8], keys[8],
		peers0123567, genesisPeerSet, validators, 1000000, 100, 10, false,
		"inmem", 10*time.Millisecond, logger, t, 12, 8, false)
	defer node8.Shutdown()

	//  Step 9 Remove Nodes 0 to 3
	leaveNode("Step 9", nodes0123[3], t)
	leaveNode("Step 9a", nodes0123[2], t)
	leaveNode("Step 9b", nodes0123[1], t)
	leaveNode("Step 9c", nodes0123[0], t)

	// New nodes array without nodes 0 to 3
	nodes5678 := nodes01235678[4:]
	gossipAndCheck(nodes5678, 13, 8, "Step 9e", false, t)

	//  Step 10 Add Node 9
	node9, nodes56789 := launchNodeAndGossip("Step 10", nodes5678, peerlist.Peers[9], keys[9],
		peers5678, genesisPeerSet, validators, 1000000, 100, 10, false,
		"inmem", 10*time.Millisecond, logger, t, 15, 8, false)
	defer node9.Shutdown()

	t.Log("Nodes56789", nodes56789)

	t.Log("Final Step")

}

/*******************************************************************************
HELPERS
*******************************************************************************/

func launchNodeAndGossip(
	msg string,
	nodeSet []*Node,
	peer *peers.Peer,
	k *ecdsa.PrivateKey,
	peers *peers.PeerSet,
	genesisPeers *peers.PeerSet,
	validators *peers.PeerSet,
	cacheSize,
	syncLimit int,
	joinTimeoutSeconds time.Duration,
	enableSyncLimit bool,
	storeType string,
	heartbeatTimeout time.Duration,
	logger *logrus.Logger,
	t *testing.T,
	targetBlockInc int,
	gossipTimeOutSeconds time.Duration,
	checkFrames bool) (node *Node, nodes []*Node) {

	t.Log(msg)

	node = newNode(peer, k, peers, genesisPeers, validators, cacheSize, syncLimit, joinTimeoutSeconds,
		enableSyncLimit, storeType, heartbeatTimeout, logger, t)

	nodes = append(append([]*Node{}, nodeSet...), node)
	node.RunAsync(true)

	gossipAndCheck(nodes, targetBlockInc, gossipTimeOutSeconds, msg, checkFrames, t)

	return node, nodes
}

func gossipAndCheck(nodes []*Node, targetBlockInc int, timeOutSeconds time.Duration, msg string, checkFrame bool,
	t *testing.T) {

	t.Log("gossipAndCheck " + msg)
	target := nodes[0].core.hg.Store.LastBlockIndex() + 1
	target += targetBlockInc

	err := gossip(nodes, target, false, timeOutSeconds*time.Second)
	if err != nil {
		t.Log("Fatal Error "+msg, err)
		t.Fatal(err)
	}
	checkGossip(nodes, target, t)

	if checkFrame {
		checkFrames(nodes, 0, t)
	}
}

func leaveNode(msg string, node *Node, t *testing.T) {
	t.Log(msg)
	err := node.Leave()
	if err != nil {
		t.Log("Fatal Error "+msg, err)
		t.Fatal(msg+" Leave", err)
	}
}
