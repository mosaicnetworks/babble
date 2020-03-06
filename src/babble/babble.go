package babble

import (
	"fmt"
	"os"
	"time"

	"github.com/mosaicnetworks/babble/src/config"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	h "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/mosaicnetworks/babble/src/service"
	"github.com/sirupsen/logrus"
)

// Babble is a struct containing the key parts of a babble node
type Babble struct {
	Config       *config.Config
	Node         *node.Node
	Transport    net.Transport
	Store        h.Store
	Peers        *peers.PeerSet
	GenesisPeers *peers.PeerSet
	Service      *service.Service
	logger       *logrus.Entry
}

// NewBabble is a factory method to produce
// a Babble instance.
func NewBabble(c *config.Config) *Babble {
	engine := &Babble{
		Config: c,
		logger: c.Logger(),
	}

	return engine
}

// Init initialises the babble engine
func (b *Babble) Init() error {

	b.logger.Debug("validateConfig")
	if err := b.validateConfig(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() validateConfig")
	}

	b.logger.Debug("initKey")
	if err := b.initKey(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() initKey")
		return err
	}

	b.logger.Debug("initPeers")
	if err := b.initPeers(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() initPeers")
		return err
	}

	b.logger.Debug("initStore")
	if err := b.initStore(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() initStore")
		return err
	}

	b.logger.Debug("initTransport")
	if err := b.initTransport(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() initTransport")
		return err
	}

	b.logger.Debug("initNode")
	if err := b.initNode(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() initNode")
		return err
	}

	b.logger.Debug("initService")
	if err := b.initService(); err != nil {
		b.logger.WithError(err).Error("babble.go:Init() initService")
		return err
	}

	return nil
}

// Run starts the Babble Node running
func (b *Babble) Run() {
	if b.Service != nil && b.Config.ServiceAddr != "" {
		go b.Service.Serve()
	}

	b.Node.Run(true)
}

func (b *Babble) validateConfig() error {
	// If --datadir was explicitely set, but not --db, the following line will
	// update the default database dir to be inside the new datadir
	b.Config.SetDataDir(b.Config.DataDir)

	logFields := logrus.Fields{
		"babble.DataDir":          b.Config.DataDir,
		"babble.BindAddr":         b.Config.BindAddr,
		"babble.AdvertiseAddr":    b.Config.AdvertiseAddr,
		"babble.ServiceAddr":      b.Config.ServiceAddr,
		"babble.NoService":        b.Config.NoService,
		"babble.MaxPool":          b.Config.MaxPool,
		"babble.Store":            b.Config.Store,
		"babble.LoadPeers":        b.Config.LoadPeers,
		"babble.LogLevel":         b.Config.LogLevel,
		"babble.Moniker":          b.Config.Moniker,
		"babble.HeartbeatTimeout": b.Config.HeartbeatTimeout,
		"babble.TCPTimeout":       b.Config.TCPTimeout,
		"babble.JoinTimeout":      b.Config.JoinTimeout,
		"babble.CacheSize":        b.Config.CacheSize,
		"babble.SyncLimit":        b.Config.SyncLimit,
		"babble.EnableFastSync":   b.Config.EnableFastSync,
		"babble.MaintenanceMode":  b.Config.MaintenanceMode,
		"babble.SuspendLimit":     b.Config.SuspendLimit,
	}

	// Maintenance-mode only works with bootstrap
	if b.Config.MaintenanceMode {
		b.logger.Debug("Config --maintenance-mode => --bootstrap")
		b.Config.Bootstrap = true
	}

	// Bootstrap only works with store
	if b.Config.Bootstrap {
		b.logger.Debug("Config --boostrap => --store")
		b.Config.Store = true
	}

	if b.Config.Store {
		logFields["babble.DatabaseDir"] = b.Config.DatabaseDir
		logFields["babble.Bootstrap"] = b.Config.Bootstrap
	}

	// SlowHeartbeat cannot be less than Heartbeat
	if b.Config.SlowHeartbeatTimeout < b.Config.HeartbeatTimeout {
		b.logger.Debugf("SlowHeartbeatTimeout (%v) cannot be less than Heartbeat (%v)",
			b.Config.SlowHeartbeatTimeout,
			b.Config.HeartbeatTimeout)
		b.Config.SlowHeartbeatTimeout = b.Config.HeartbeatTimeout
	}
	logFields["babble.SlowHeartbeatTimeout"] = b.Config.SlowHeartbeatTimeout

	b.logger.WithFields(logFields).Debug("Config")

	return nil
}

func (b *Babble) initTransport() error {
	// XXX
	if b.Config.WebRTC {
		fmt.Println("XXX init webrtc, with signal addr ", b.Config.SignalAddr)
		signal, err := wamp.NewClient(
			b.Config.SignalAddr,
			"office", // XXX
			keys.PublicKeyHex(&b.Config.Key.PublicKey)) //XXX

		if err != nil {
			fmt.Println("XXX wtf")
			return err
		}

		webRTCTransport, err := net.NewWebRTCTransport(
			signal,
			1,
			time.Second,
			2*time.Second,
			b.logger.WithField("component", "signal_client"))

		if err != nil {
			return err
		}

		b.Transport = webRTCTransport
	} else {
		tcpTransport, err := net.NewTCPTransport(
			b.Config.BindAddr,
			b.Config.AdvertiseAddr,
			b.Config.MaxPool,
			b.Config.TCPTimeout,
			b.Config.JoinTimeout,
			b.Config.Logger(),
		)

		if err != nil {
			return err
		}

		b.Transport = tcpTransport
	}

	return nil
}

func (b *Babble) initPeers() error {
	if !b.Config.LoadPeers {
		if b.Peers == nil {
			return fmt.Errorf("LoadPeers false, but babble.Peers is nil")
		}

		if b.GenesisPeers == nil {
			return fmt.Errorf("LoadPeers false, but babble.GenesisPeers is nil")
		}

		b.logger.Debug("LoadPeers is false. Skipped loading peers.")
		return nil
	}

	peerStore := peers.NewJSONPeerSet(b.Config.DataDir, true)

	participants, err := peerStore.PeerSet()
	if err != nil {
		return err
	}

	b.Peers = participants

	b.logger.Debug("Loaded Peers")
	//	b.logger.WithField("*participants", *participants).Info("Created Peers")

	// Set Genesis Peer Set from peers.genesis.json
	genesisPeerStore := peers.NewJSONPeerSet(b.Config.DataDir, false)

	genesisParticipants, err := genesisPeerStore.PeerSet()
	if err != nil { // If there is any error, the current peer set is used as the genesis peer set
		b.logger.Debugf("could not read peers.genesis.json: %v", err)
		b.GenesisPeers = participants
	} else {
		b.GenesisPeers = genesisParticipants
	}

	return nil
}

func (b *Babble) initStore() error {
	if !b.Config.Store {
		b.logger.Debug("Creating InmemStore")
		b.Store = h.NewInmemStore(b.Config.CacheSize)
	} else {
		dbPath := b.Config.DatabaseDir

		b.logger.WithField("path", dbPath).Debug("Creating BadgerStore")

		if !b.Config.Bootstrap {
			b.logger.Debug("No Bootstrap")

			backup := backupFileName(dbPath)

			err := os.Rename(dbPath, backup)

			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				b.logger.Debug("Nothing to backup")
			} else {
				b.logger.WithField("path", backup).Debug("Created backup")
			}
		}

		b.logger.WithField("path", dbPath).Debug("Opening BadgerStore")

		dbStore, err := h.NewBadgerStore(
			b.Config.CacheSize,
			dbPath,
			b.Config.MaintenanceMode,
			b.logger)
		if err != nil {
			return err
		}

		b.Store = dbStore
	}

	return nil
}

func (b *Babble) initKey() error {
	if b.Config.Key == nil {
		simpleKeyfile := keys.NewSimpleKeyfile(b.Config.Keyfile())

		privKey, err := simpleKeyfile.ReadKey()
		if err != nil {
			b.logger.Errorf("Error reading private key from file: %v", err)
		}

		b.Config.Key = privKey
	}
	return nil
}

func (b *Babble) initNode() error {

	validator := node.NewValidator(b.Config.Key, b.Config.Moniker)

	p, ok := b.Peers.ByID[validator.ID()]
	if ok {
		if p.Moniker != validator.Moniker {
			b.logger.WithFields(logrus.Fields{
				"json_moniker": p.Moniker,
				"cli_moniker":  validator.Moniker,
			}).Debugf("Using moniker from peers.json file")
			validator.Moniker = p.Moniker
		}
	}

	b.Config.Logger().WithFields(logrus.Fields{
		"genesis_peers": len(b.GenesisPeers.Peers),
		"peers":         len(b.Peers.Peers),
		"id":            validator.ID(),
		"moniker":       validator.Moniker,
	}).Debug("PARTICIPANTS")

	b.Node = node.NewNode(
		b.Config,
		validator,
		b.Peers,
		b.GenesisPeers,
		b.Store,
		b.Transport,
		b.Config.Proxy,
	)

	return b.Node.Init()
}

func (b *Babble) initService() error {
	if !b.Config.NoService {
		b.Service = service.NewService(b.Config.ServiceAddr, b.Node, b.Config.Logger())
	}
	return nil
}

// backupFileName implements the naming convention for database backups:
// badger_db--UTC--<created_at UTC ISO8601>
func backupFileName(base string) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("%s--UTC--%s", base, toISO8601(ts))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}
