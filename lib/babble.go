package babble

import (
	"fmt"

	"github.com/mosaicnetworks/babble/crypto"
	h "github.com/mosaicnetworks/babble/hashgraph"
	"github.com/mosaicnetworks/babble/net"
	"github.com/mosaicnetworks/babble/node"
	"github.com/mosaicnetworks/babble/peers"
	"github.com/sirupsen/logrus"
)

type Babble struct {
	Config    *BabbleConfig
	Node      *node.Node
	Transport net.Transport
	Store     h.Store
	Peers     *peers.Peers
}

func NewBabble(config *BabbleConfig) *Babble {
	engine := &Babble{
		Config: config,
	}

	return engine
}

func (b *Babble) initTransport() error {
	transport, err := net.NewTCPTransport(
		b.Config.BindAddr,
		nil,
		b.Config.MaxPool,
		b.Config.NodeConfig.HeartbeatTimeout,
		b.Config.Logger,
	)

	if err != nil {
		return err
	}

	b.Transport = transport

	return nil
}

func (b *Babble) initPeers() error {
	if !b.Config.LoadPeers {
		return nil
	}

	peerStore := peers.NewJSONPeers(b.Config.DataDir)

	participants, err := peerStore.Peers()

	if err != nil {
		return err
	}

	if participants.Len() < 2 {
		return fmt.Errorf("peers.json should define at least two peers")
	}

	b.Peers = participants

	return nil
}

func (b *Babble) initStore() error {
	if b.Config.StorePath == "" {
		b.Store = h.NewInmemStore(b.Peers, b.Config.NodeConfig.CacheSize)
	} else {
		var err error

		b.Store, err = h.LoadOrCreateBadgerStore(b.Peers, b.Config.NodeConfig.CacheSize, b.Config.StorePath)

		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Babble) initNode() error {
	key := b.Config.Key

	if b.Config.Key == nil {
		pemKey := crypto.NewPemKey(b.Config.DataDir)

		privKey, err := pemKey.ReadKey()

		if err != nil {
			return err
		}

		key = privKey
	}

	nodePub := fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey))
	nodeID := b.Peers.ByPubKey[nodePub].ID

	b.Node = node.NewNode(
		b.Config.NodeConfig,
		nodeID,
		key,
		b.Peers,
		b.Store,
		b.Transport,
		b.Config.Proxy,
	)

	if err := b.Node.Init(); err != nil {
		return fmt.Errorf("failed to initialize node: %s", err)
	}

	return nil
}

func (b *Babble) Init() error {
	if b.Config.Logger == nil {
		b.Config.Logger = logrus.New()
	}

	if err := b.initPeers(); err != nil {
		return err
	}

	if err := b.initTransport(); err != nil {
		return err
	}

	if err := b.initStore(); err != nil {
		return err
	}

	if err := b.initNode(); err != nil {
		return err
	}

	return nil
}

func (b *Babble) Run() {
	b.Node.Run(true)
}

func Keygen() (*crypto.PemDump, error) {
	return crypto.GeneratePemKey()
}
