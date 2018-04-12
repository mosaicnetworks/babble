package mobile

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/babbleio/babble/crypto"
	hg "github.com/babbleio/babble/hashgraph"
	"github.com/babbleio/babble/net"
	"github.com/babbleio/babble/node"
	"github.com/babbleio/babble/proxy"
	"github.com/sirupsen/logrus"
)

type Node struct {
	nodeID int
	node   *node.Node
	proxy  proxy.AppProxy
	logger *logrus.Logger
}

// New initializes Node struct
func New(nodeAddr string,
	peers string,
	privKey string,
	commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	config *MobileConfig) *Node {

	logger := initLogger()

	var netPeers []net.Peer
	err := json.Unmarshal([]byte(peers), &netPeers)
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("l37: %s. %s", err.Error(), peers))
		return nil
	}

	conf := node.NewConfig(
		time.Duration(config.HeartBeat)*time.Millisecond,
		time.Duration(1000)*time.Millisecond,
		config.CacheSize,
		config.SyncLimit,
		"inmem",
		"",
		logger)

	maxPool := config.MaxPool

	pemKey := &crypto.PemKey{}
	key, err := pemKey.ReadKeyFromBuf([]byte(privKey))
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("l55: %s", "Failed to read private key"))
		return nil
	}

	sort.Sort(net.ByPubKey(netPeers))
	pmap := make(map[string]int)
	for i, p := range netPeers {
		pmap[p.PubKeyHex] = i
	}

	logger.WithField("node_addr", nodeAddr).Debug("Node Address")

	//Find the ID of this node
	nodePub := fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey))
	nodeID := pmap[nodePub]

	logger.WithFields(logrus.Fields{
		"pmap": pmap,
		"id":   nodeID,
	}).Debug("PARTICIPANTS")

	needBootstrap := false
	store := hg.NewInmemStore(pmap, conf.CacheSize)

	trans, err := net.NewTCPTransport(
		nodeAddr, nil, maxPool, conf.TCPTimeout, logger)
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("l82: %s", err.Error()))
		return nil
	}

	var prox proxy.AppProxy
	prox = newMobileAppProxy(commitHandler, exceptionHandler, logger)

	node := node.NewNode(conf, nodeID, key, netPeers, store, trans, prox)
	if err := node.Init(needBootstrap); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("l91 %s", fmt.Sprintf("failed to initialize node: %s", err)))
	}

	return &Node{
		node:   node,
		proxy:  prox,
		nodeID: nodeID,
		logger: logger,
	}
}

func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	return logger
}

func (n *Node) Run(async bool) {
	if async {
		n.node.RunAsync(true)
	} else {
		n.node.Run(true)
	}
}

func (n *Node) Shutdown() {
	n.node.Shutdown()
}

func (n *Node) SubmitTx(tx []byte) {
	n.proxy.SubmitCh() <- tx
}
