package mobile

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/mosaicnetworks/babble/crypto"
	hg "github.com/mosaicnetworks/babble/hashgraph"
	"github.com/mosaicnetworks/babble/net"
	"github.com/mosaicnetworks/babble/node"
	"github.com/mosaicnetworks/babble/proxy"
	"github.com/sirupsen/logrus"
)

type Node struct {
	nodeID int
	node   *node.Node
	proxy  proxy.AppProxy
	logger *logrus.Logger
}

// New initializes Node struct
func New(privKey string,
	nodeAddr string,
	peers string,
	commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	config *MobileConfig) *Node {

	logger := initLogger()

	logger.WithFields(logrus.Fields{
		"nodeAddr": nodeAddr,
		"peers":    peers,
		"config":   fmt.Sprintf("%v", config),
	}).Debug("New Mobile Node")

	//Check private key
	pemKey := &crypto.PemKey{}
	key, err := pemKey.ReadKeyFromBuf([]byte(privKey))
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Failed to read private key: %s", err))
		return nil
	}

	//Check peers
	var netPeers []net.Peer
	if err := json.Unmarshal([]byte(peers), &netPeers); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Unmarshalling json peers: %s. %s", err.Error(), peers))
		return nil
	}

	// There should be at least two peers
	if len(peers) < 2 {
		exceptionHandler.OnException(fmt.Sprintf("Should define at least two peers"))
		return nil
	}

	sort.Sort(net.ByPubKey(netPeers))
	pmap := make(map[string]int)
	for i, p := range netPeers {
		pmap[p.PubKeyHex] = i
	}

	//Find the ID of this node
	nodePub := fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey))
	nodeID := pmap[nodePub]

	logger.WithFields(logrus.Fields{
		"pmap": pmap,
		"id":   nodeID,
	}).Debug("PARTICIPANTS")

	conf := node.NewConfig(
		time.Duration(config.Heartbeat)*time.Millisecond,
		time.Duration(config.TCPTimeout)*time.Millisecond,
		config.CacheSize,
		config.SyncLimit,
		config.StoreType,
		config.StorePath,
		logger)

	//Instantiate the Store (inmem or badger)
	var store hg.Store
	var needBootstrap bool
	switch conf.StoreType {
	case "inmem":
		store = hg.NewInmemStore(pmap, conf.CacheSize)
	case "badger":
		//If the file already exists, load and bootstrap the store using the file
		if _, err := os.Stat(conf.StorePath); err == nil {
			logger.Debug("loading badger store from existing database")
			store, err = hg.LoadBadgerStore(conf.CacheSize, conf.StorePath)
			if err != nil {
				exceptionHandler.OnException(fmt.Sprintf("failed to load BadgerStore from existing file: %s", err))
				return nil
			}
			needBootstrap = true
		} else {
			//Otherwise create a new one
			logger.Debug("creating new badger store from fresh database")
			store, err = hg.NewBadgerStore(pmap, conf.CacheSize, conf.StorePath)
			if err != nil {
				exceptionHandler.OnException(fmt.Sprintf("failed to create new BadgerStore: %s", err))
				return nil
			}
		}
	default:
		exceptionHandler.OnException(fmt.Sprintf("Invalid StoreType: %s", conf.StoreType))
		return nil
	}

	trans, err := net.NewTCPTransport(
		nodeAddr, nil, config.MaxPool, conf.TCPTimeout, logger)
	if err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Creating TCP Transport: %s", err.Error()))
		return nil
	}

	var prox proxy.AppProxy
	prox = newMobileAppProxy(commitHandler, exceptionHandler, logger)

	node := node.NewNode(conf, nodeID, key, netPeers, store, trans, prox)
	if err := node.Init(needBootstrap); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Initializing node: %s", err))
		return nil
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
	//have to make a copy or the tx will be garbage collected and weird stuff
	//happens in transaction pool
	t := make([]byte, len(tx), len(tx))
	copy(t, tx)
	n.proxy.SubmitCh() <- t
}
