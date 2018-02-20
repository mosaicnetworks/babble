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
	aproxy "github.com/babbleio/babble/proxy/app"
	// "github.com/babbleio/babble/crypto"
	// hg "github.com/babbleio/babble/hashgraph"
	// "github.com/babbleio/babble/net"
	// node "github.com/babbleio/babble/node"
	// "github.com/babbleio/babble/proxy"
	// aproxy "github.com/babbleio/babble/proxy/app"
	// "github.com/sirupsen/logrus"
)

type Node struct {
	node *node.Node
}

// New initializes Node struct
func New(peers string, privKey string, events *EventHandler, config *Config) *Node {

	sub := &Subscription{events: events}

	var netPeers []net.Peer
	err := json.Unmarshal([]byte(peers), &netPeers)

	if err != nil {
		sub.message(err.Error())
	} else {
		sub.message(netPeers[0].NetAddr)
	}

	logger := Logger()

	conf := node.NewConfig(
		time.Duration(config.HeartBeat)*time.Millisecond,
		time.Duration(1000)*time.Millisecond,
		config.CacheSize,
		config.SyncLimit,
		"inmem",
		"",
		logger)

	maxPool := config.MaxPool

	addr := "127.0.0.1:1337"
	clientAddress := "127.0.0.1:1338"
	proxyAddress := "127.0.0.1:"

	pemKey := &crypto.PemKey{}
	key, err := pemKey.ReadKeyFromBuf([]byte(privKey))
	if err != nil {
		sub.message("Fail to read private key")
		return nil
	}

	store := hg.NewInmemStore(nil, conf.CacheSize)

	trans, err := net.NewTCPTransport(
		addr, nil, maxPool, conf.TCPTimeout, logger)
	if err != nil {
		sub.message("Fail to create TCP")
		return nil
	}

	var prox proxy.AppProxy
	//prox = aproxy.NewInmemAppProxy(logger)
	prox = aproxy.NewSocketAppProxy(clientAddress, proxyAddress,
		conf.TCPTimeout, logger)

	needBootstrap := false

	//netPeers := make([]net.Peer, len(peers))

	// for i, p := range peers {
	// 	netPeers[i] = p.toNetPeer()
	// }

	sort.Sort(net.ByPubKey(netPeers))
	pmap := make(map[string]int)
	for i, p := range netPeers {
		pmap[p.PubKeyHex] = i
	}

	//Find the ID of this node
	nodePub := fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey))
	nodeID := pmap[nodePub]

	node := node.NewNode(conf, nodeID, key, netPeers, store, trans, prox)
	if err := node.Init(needBootstrap); err != nil {
		sub.message(err.Error())
	}

	//node.Run(true)

	return &Node{node: node}
}

func Hello(input string) string {
	return fmt.Sprintf("Hello, %s!", input)
}

func (n *Node) Run() {
	n.node.Run(true)
}
