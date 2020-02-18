package mobile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/mosaicnetworks/babble/src/babble"
	"github.com/mosaicnetworks/babble/src/config"
	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/mosaicnetworks/babble/src/proxy/inmem"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Node is the object that mobile consumers interact with.
type Node struct {
	nodeID uint32
	node   *node.Node
	proxy  proxy.AppProxy
	logger *logrus.Entry
}

// New initializes Node struct
func New(
	commitHandler CommitHandler,
	stateChangeHandler StateChangeHandler,
	exceptionHandler ExceptionHandler,
	configDir string,
) *Node {

	babbleConfig := config.NewDefaultConfig()
	v := viper.New()

	v.SetConfigName("babble")  // name of config file (without extension)
	v.AddConfigPath(configDir) // search root directory

	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err == nil {
		babbleConfig.Logger().Debugf("Using config file: %s", v.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		babbleConfig.Logger().Debugf("No config file found in: %s", filepath.Join(configDir, "babble.toml"))
	} else {
		babbleConfig.Logger().Errorf("Error loading config file: %v", err)
		return nil
	}

	if err := v.Unmarshal(babbleConfig); err != nil {
		babbleConfig.Logger().Errorf("Error marshalling config file: %v", err)
		return nil
	}

	babbleConfig.Logger().WithFields(logrus.Fields{
		"config": fmt.Sprintf("%v", babbleConfig),
	}).Debug("New Mobile Node")

	// mobileApp implements the ProxyHandler interface, and we use it to
	// instantiate an InmemProxy
	mobileApp := newMobileApp(
		commitHandler,
		stateChangeHandler,
		exceptionHandler,
		babbleConfig.Logger())
	babbleConfig.Proxy = inmem.NewInmemProxy(mobileApp, babbleConfig.Logger())

	engine := babble.NewBabble(babbleConfig)

	if err := engine.Init(); err != nil {
		exceptionHandler.OnException(fmt.Sprintf("Cannot initialize engine: %s", err))
		return nil
	}

	return &Node{
		node:   engine.Node,
		proxy:  babbleConfig.Proxy,
		nodeID: engine.Node.GetID(),
		logger: babbleConfig.Logger(),
	}
}

// Run runs the Babble node
func (n *Node) Run(async bool) {
	if async {
		n.node.RunAsync(true)
	} else {
		n.node.Run(true)
	}
}

// Leave instructs the node to leave politely (get removed from validator-set)
// before shutting down.
func (n *Node) Leave() {
	n.node.Leave()
}

// Shutdown shutsdown the node without requesting to be removed from the
// validator-set
func (n *Node) Shutdown() {
	n.node.Shutdown()
}

// SubmitTx submits a transaction to Babble for consensus ordering.
func (n *Node) SubmitTx(tx []byte) {
	// have to make a copy or the tx will be garbage collected and weird stuff
	// happens in transaction pool
	t := make([]byte, len(tx), len(tx))
	copy(t, tx)
	n.proxy.SubmitCh() <- t
}

// GetPeers returns the current list of peers.
func (n *Node) GetPeers() string {
	peers := n.node.GetPeers()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peers); err != nil {
		return ""
	}

	return buf.String()
}

// GetGenesisPeers returns the genesis peers.
func (n *Node) GetGenesisPeers() string {
	peers, err := n.node.GetValidatorSet(0)

	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peers); err != nil {
		return ""
	}

	return buf.String()
}

// GetStats returns consensus stats.
func (n *Node) GetStats() string {
	stats := n.node.GetStats()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(stats); err != nil {
		return ""
	}

	return buf.String()
}
