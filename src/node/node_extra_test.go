// +build !unit

package node

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"

	"github.com/sirupsen/logrus"

	bkeys "github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*

Some complicated tests of dynamic membership. They are quite long so we don't
always run them. The '+build !unit' at the top of this file indicates that these
tests will not run when the '-tags=unit' is passed to the go test command
(cf makefile).

fast-sync disabled

*/

func TestJoinLateExtra(t *testing.T) {

	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	keys, peerSet := initPeers(t, 4)

	initialNodes := initNodes(keys, peerSet, peerSet, 400, 400, 5, false, "badger", 30*time.Millisecond, common.TestLogLevel, t)
	defer shutdownNodes(initialNodes)

	target := 100
	err := gossip(initialNodes, target, false, 10*time.Second)
	if err != nil {
		t.Fatalf("Fatal Error: %v", err)
	}
	checkGossip(initialNodes, 0, t)

	key, _ := bkeys.GenerateECDSAKey()
	peer := peers.NewPeer(
		bkeys.PublicKeyHex(&key.PublicKey),
		fmt.Sprint("127.0.0.1:4244"),
		"monika",
	)

	newNode := newNode(peer, key, peerSet, peerSet, 400, 400, 5, false, "badger", 30*time.Millisecond, common.TestLogLevel, t)
	defer newNode.Shutdown()

	newNode.RunAsync(true)

	nodes := append(initialNodes, newNode)

	//defer drawGraphs(nodes, t)

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes, secondTarget, 20*time.Second)
	if err != nil {
		t.Fatalf("Fatal Error: %v", err)
	}

	start := newNode.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
	checkPeerSets(nodes, t)
	verifyNewPeerSet(nodes, newNode.core.AcceptedRound, 5, t)

	time.Sleep(2 * time.Second)
}

func TestSuccessiveJoinRequestExtra(t *testing.T) {

	keys, peerSet := initPeers(t, 1)
	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	node0 := newNode(peerSet.Peers[0], keys[0], peerSet, genesisPeerSet, 10000, 400, 5, false, "inmem", 30*time.Millisecond, common.TestLogLevel, t)
	defer node0.Shutdown()
	node0.RunAsync(true)

	nodes := []*Node{node0}

	target := 10
	for i := 1; i <= 3; i++ {
		peerSet := peers.NewPeerSet(node0.GetPeers())

		key, _ := bkeys.GenerateECDSAKey()
		peer := peers.NewPeer(
			bkeys.PublicKeyHex(&key.PublicKey),
			fmt.Sprintf("127.0.0.1:%d", 4240+i),
			fmt.Sprintf("monika%d", i),
		)
		newNode := newNode(peer, key, peerSet, genesisPeerSet, 1000, 400, 5, false, "inmem", 30*time.Millisecond, common.TestLogLevel, t)

		t.Logf("starting new node %d, %d", i, peer.ID())
		defer newNode.Shutdown()
		newNode.RunAsync(true)

		nodes = append(nodes, newNode)

		//Gossip some more
		err := bombardAndWait(nodes, target, 20*time.Second)
		if err != nil {
			t.Error("Fatal Error in TestSuccessiveJoinRequestExtra", err)
			t.Fatal(err)
		}

		start := newNode.core.hg.FirstConsensusRound
		checkGossip(nodes, *start, t)
		checkPeerSets(nodes, t)

		target = target + 10
	}
}

func TestSuccessiveLeaveRequestExtra(t *testing.T) {
	n := 4

	keys, peerSet := initPeers(t, n)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, 1000, 10000, 20, false, "inmem", 30*time.Millisecond, common.TestLogLevel, t)
	defer shutdownNodes(nodes)

	target := 0

	f := func() {
		t.Logf("SUCCESSIVE LEAVE n=%d", n)
		//defer drawGraphs(nodes, t)
		target += 10
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
		target += 10
		err = bombardAndWait(nodes, target, 16*time.Second)
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
	keys, peerSet := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	nodes := initNodes(keys, peerSet, genesisPeerSet, 10000, 1000, 5, false, "inmem", 30*time.Millisecond, common.TestLogLevel, t)
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
	err = bombardAndWait(nodes[0:2], secondTarget, 16*time.Second)
	if err != nil {
		t.Error("Fatal Error 4", err)
		t.Fatal(err)
	}
	checkGossip(nodes[0:2], 0, t)
	checkPeerSets(nodes[0:2], t)
}

func TestJoinLeaveRequestExtra(t *testing.T) {
	keys, peerSet := initPeers(t, 4)

	genesisPeerSet := clonePeerSet(t, peerSet.Peers)

	// N.B. Info level logs to keep the log size manageable
	nodes := initNodes(keys, peerSet, genesisPeerSet, 1000, 10000, 5, false, "inmem", 30*time.Millisecond, logrus.InfoLevel, t)
	//defer drawGraphs(nodes, t)

	target := 15
	err := gossip(nodes, target, false, 2*time.Second)
	if err != nil {
		t.Error("Fatal Error", err)
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	leavingNode := nodes[3]
	defer leavingNode.Shutdown()

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
	newNode := newNode(peer, key, peerSet, genesisPeerSet, 10000, 200, 5, false, "inmem", 30*time.Millisecond, common.TestLogLevel, t)
	defer newNode.Shutdown()

	newNode.RunAsync(true)

	// replace leaving node with new node
	nodes[3] = newNode

	t.Log("Node 3 Created")
	newNode.RunAsync(true)

	//Gossip some more
	secondTarget := target + 12
	err = bombardAndWait(nodes, secondTarget, 16*time.Second)
	if err != nil {
		t.Error("Fatal Error 3", err)
		t.Fatal(err)
	}

	start := newNode.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
	checkPeerSets(nodes, t)
}

// TestAddingAndRemovingPeers is a complex test. The broad brush outline of the
// process is as follows:
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
	keys, peerlist := initPeers(t, 10)
	// Prebuild Peers
	peers01234 := clonePeerSet(t, peerlist.Peers[0:5])
	peers0123 := clonePeerSet(t, peerlist.Peers[0:4])
	peers01235 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:6]...))   // Step 4
	peers012356 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:7]...))  // Step 6
	peers0123567 := clonePeerSet(t, append(append([]*peers.Peer{}, peerlist.Peers[0:4]...), peerlist.Peers[5:8]...)) // Step 8
	peers5678 := clonePeerSet(t, peerlist.Peers[5:9])                                                                // Step 10
	genesisPeerSet := clonePeerSet(t, peers01234.Peers)                                                              // Step 10

	t.Log("Step 1")
	// N.B. Info level logs to keep the log size manageable
	nodes01234 := initNodes(keys[0:5], peers01234, genesisPeerSet, 10000, 400, 15, false, "inmem", 30*time.Millisecond, logrus.InfoLevel, t) //make cache high to draw graphs

	// Step 1b - gossip and build history
	gossipAndCheck(nodes01234, 20, 8, "Step 1b", false, t)
	checkPeerSets(nodes01234, t)

	// Step 2 - Node 4 leaves
	leaveNode("Step 2", nodes01234[4], t)

	// New nodes array without node 4
	nodes0123 := nodes01234[0:4]
	checkPeerSets(nodes0123, t)

	// Step 3 - More history
	gossipAndCheck(nodes0123, 30, 8, "Step 3", false, t)

	// Step 4 Add a new validator (node 5) and sync without using fast sync
	node5, nodes01235 := launchNodeAndGossip("Step 4 and 5", nodes0123, peerlist.Peers[5], keys[5],
		peers0123, genesisPeerSet, 10000, 100, 10, false,
		"inmem", 30*time.Millisecond, t, 20, 8, true, logrus.InfoLevel)
	defer node5.Shutdown()

	// Step 6 Add another validator (node 6) and sync without fast sync
	// Step 7 Add more history and check that all peers have the same state

	node6, nodes012356 := launchNodeAndGossip("Step 6 and 7", nodes01235, peerlist.Peers[6], keys[6],
		peers01235, genesisPeerSet, 10000, 100, 10, false,
		"inmem", 30*time.Millisecond, t, 15, 8, true, logrus.InfoLevel)
	defer node6.Shutdown()

	//  Step 8 Add Node 7, 8

	node7, nodes0123567 := launchNodeAndGossip("Step 8", nodes012356, peerlist.Peers[7], keys[7],
		peers012356, genesisPeerSet, 10000, 100, 10, false,
		"inmem", 30*time.Millisecond, t, 14, 8, false, logrus.InfoLevel)
	defer node7.Shutdown()

	node8, nodes01235678 := launchNodeAndGossip("Step 8b", nodes0123567, peerlist.Peers[8], keys[8],
		peers0123567, genesisPeerSet, 10000, 100, 10, false,
		"inmem", 30*time.Millisecond, t, 12, 8, false, logrus.InfoLevel)
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
		peers5678, genesisPeerSet, 10000, 100, 10, false,
		"inmem", 30*time.Millisecond, t, 15, 8, false, logrus.InfoLevel)
	defer node9.Shutdown()

	t.Log("Nodes56789", nodes56789)
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
	cacheSize,
	syncLimit int,
	joinTimeoutSeconds time.Duration,
	enableSyncLimit bool,
	storeType string,
	heartbeatTimeout time.Duration,
	t *testing.T,
	targetBlockInc int,
	gossipTimeOutSeconds time.Duration,
	checkFrames bool,
	logLevel logrus.Level) (node *Node, nodes []*Node) {

	t.Log(msg)

	node = newNode(peer,
		k,
		peers,
		genesisPeers,
		cacheSize,
		syncLimit,
		joinTimeoutSeconds,
		enableSyncLimit,
		storeType,
		heartbeatTimeout,
		logLevel,
		t)

	nodes = append(append([]*Node{}, nodeSet...), node)
	node.RunAsync(true)

	catchUpBlock, _ := strconv.ParseInt(nodes[0].GetStats()["last_block_index"], 10, 64)

	// Let the new node catch up.
	for {
		time.Sleep(1 * time.Second)
		stats := node.GetStats()
		nodeState := stats["state"]
		lastBlock, _ := strconv.ParseInt(stats["last_block_index"], 10, 64)
		t.Logf("launchNodeAndGossip(%s): %d of %d ", nodeState, catchUpBlock, lastBlock)

		if lastBlock >= catchUpBlock {
			break
		}
	}

	time.Sleep(1 * time.Second)
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
