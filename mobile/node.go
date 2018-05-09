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
	sub    *Subscription
	logger *logrus.Logger
}

// New initializes Node struct
func New(node_addr string, peers string, privKey string, events *EventHandler, config *Config) *Node {

	logger := Logger()
	sub := &Subscription{events: events}

	var netPeers []net.Peer
	err := json.Unmarshal([]byte(peers), &netPeers)
	if err != nil {
		sub.error(err.Error())
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
		sub.error("Fail to read private key")
		return nil
	}

	sort.Sort(net.ByPubKey(netPeers))
	pmap := make(map[string]int)
	for i, p := range netPeers {
		pmap[p.PubKeyHex] = i
	}

	logger.WithField("node_addr", node_addr).Debug("Node Address")

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
		node_addr, nil, maxPool, conf.TCPTimeout, logger)
	if err != nil {
		sub.error(err.Error())
		return nil
	}

	var prox proxy.AppProxy
	prox = NewAppProxy(sub, logger)

	node := node.NewNode(conf, nodeID, key, netPeers, store, trans, prox)
	if err := node.Init(needBootstrap); err != nil {
		sub.error(fmt.Sprintf("failed to initialize node: %s", err))
	}

	return &Node{
		node:   node,
		proxy:  prox,
		sub:    sub,
		nodeID: nodeID,
		logger: logger,
	}
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

func (n *Node) pack(tx []byte) []byte {
	var buf []byte

	buf = append(buf, '*', 'b')            // prefix
	buf = append(buf, byte(n.nodeID&0x0F)) // node ID
	buf = append(buf, byte((n.nodeID>>8)&0x0F))
	buf = append(buf, tx...) // ball data

	n.logger.WithField("bytes", fmt.Sprintf("[% x]", buf)).Debug("Pack Transaction")

	return buf
}

func (n *Node) SubmitTx(tx []byte) {
	n.proxy.SubmitCh() <- n.pack(tx)
}
