package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/mosaicnetworks/babble/crypto"
	hg "github.com/mosaicnetworks/babble/hashgraph"
	"github.com/mosaicnetworks/babble/net"
	"github.com/mosaicnetworks/babble/node"
	"github.com/mosaicnetworks/babble/proxy"
	aproxy "github.com/mosaicnetworks/babble/proxy/app"
	"github.com/mosaicnetworks/babble/service"
	"github.com/mosaicnetworks/babble/version"
)

var (
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Directory for the configuration",
		Value: defaultDataDir(),
	}
	NodeAddressFlag = cli.StringFlag{
		Name:  "node_addr",
		Usage: "IP:Port to bind Babble",
		Value: "127.0.0.1:1337",
	}
	NoClientFlag = cli.BoolFlag{
		Name:  "no_client",
		Usage: "Run Babble with dummy in-memory App client",
	}
	ProxyAddressFlag = cli.StringFlag{
		Name:  "proxy_addr",
		Usage: "IP:Port to bind Proxy Server",
		Value: "127.0.0.1:1338",
	}
	ClientAddressFlag = cli.StringFlag{
		Name:  "client_addr",
		Usage: "IP:Port of Client App",
		Value: "127.0.0.1:1339",
	}
	ServiceAddressFlag = cli.StringFlag{
		Name:  "service_addr",
		Usage: "IP:Port of HTTP Service",
		Value: "127.0.0.1:8000",
	}
	LogLevelFlag = cli.StringFlag{
		Name:  "log_level",
		Usage: "debug, info, warn, error, fatal, panic",
		Value: "debug",
	}
	HeartbeatFlag = cli.IntFlag{
		Name:  "heartbeat",
		Usage: "Heartbeat timer milliseconds (time between gossips)",
		Value: 1000,
	}
	MaxPoolFlag = cli.IntFlag{
		Name:  "max_pool",
		Usage: "Max number of pooled connections",
		Value: 2,
	}
	TcpTimeoutFlag = cli.IntFlag{
		Name:  "tcp_timeout",
		Usage: "TCP timeout milliseconds",
		Value: 1000,
	}
	CacheSizeFlag = cli.IntFlag{
		Name:  "cache_size",
		Usage: "Number of items in LRU caches",
		Value: 500,
	}
	SyncLimitFlag = cli.IntFlag{
		Name:  "sync_limit",
		Usage: "Max number of events for sync",
		Value: 1000,
	}
	StoreFlag = cli.StringFlag{
		Name:  "store",
		Usage: "badger, inmem",
		Value: "badger",
	}
	StorePathFlag = cli.StringFlag{
		Name:  "store_path",
		Usage: "File containing the store database",
		Value: defaultBadgerDir(),
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "babble"
	app.Usage = "hashgraph consensus"
	app.HideVersion = true //there is a special command to print the version
	app.Commands = []cli.Command{
		{
			Name:   "keygen",
			Usage:  "Dump new key pair",
			Action: keygen,
		},
		{
			Name:   "run",
			Usage:  "Run node",
			Action: run,
			Flags: []cli.Flag{
				DataDirFlag,
				NodeAddressFlag,
				NoClientFlag,
				ProxyAddressFlag,
				ClientAddressFlag,
				ServiceAddressFlag,
				LogLevelFlag,
				HeartbeatFlag,
				MaxPoolFlag,
				TcpTimeoutFlag,
				CacheSizeFlag,
				SyncLimitFlag,
				StoreFlag,
				StorePathFlag,
			},
		},
		{
			Name:   "version",
			Usage:  "Show version info",
			Action: printVersion,
		},
	}
	app.Run(os.Args)
}

func keygen(c *cli.Context) error {
	pemDump, err := crypto.GeneratePemKey()
	if err != nil {
		fmt.Println("Error generating PemDump")
		os.Exit(2)
	}

	fmt.Println("PublicKey:")
	fmt.Println(pemDump.PublicKey)
	fmt.Println("PrivateKey:")
	fmt.Println(pemDump.PrivateKey)

	return nil
}

func run(c *cli.Context) error {
	logger := logrus.New()
	logger.Level = logLevel(c.String(LogLevelFlag.Name))

	datadir := c.String(DataDirFlag.Name)
	addr := c.String(NodeAddressFlag.Name)
	noclient := c.Bool(NoClientFlag.Name)
	proxyAddress := c.String(ProxyAddressFlag.Name)
	clientAddress := c.String(ClientAddressFlag.Name)
	serviceAddress := c.String(ServiceAddressFlag.Name)
	heartbeat := c.Int(HeartbeatFlag.Name)
	maxPool := c.Int(MaxPoolFlag.Name)
	tcpTimeout := c.Int(TcpTimeoutFlag.Name)
	cacheSize := c.Int(CacheSizeFlag.Name)
	syncLimit := c.Int(SyncLimitFlag.Name)
	storeType := c.String(StoreFlag.Name)
	storePath := c.String(StorePathFlag.Name)

	logger.WithFields(logrus.Fields{
		"datadir":      datadir,
		"node_addr":    addr,
		"no_client":    noclient,
		"proxy_addr":   proxyAddress,
		"client_addr":  clientAddress,
		"service_addr": serviceAddress,
		"heartbeat":    heartbeat,
		"max_pool":     maxPool,
		"tcp_timeout":  tcpTimeout,
		"cache_size":   cacheSize,
		"store":        storeType,
		"store_path":   storePath,
	}).Debug("RUN")

	conf := node.NewConfig(time.Duration(heartbeat)*time.Millisecond,
		time.Duration(tcpTimeout)*time.Millisecond,
		cacheSize, syncLimit, storeType, storePath, logger)

	// Create the PEM key
	pemKey := crypto.NewPemKey(datadir)

	// Try a read
	key, err := pemKey.ReadKey()
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	// Create the peer store
	peerStore := net.NewJSONPeers(datadir)

	// Try a read
	peers, err := peerStore.Peers()
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	// There should be at least two peers
	if len(peers) < 2 {
		return cli.NewExitError("peers.json should define at least two peers", 1)
	}

	//Sort peers by public key and assign them an int ID
	//Every participant in the network will run this and assign the same IDs
	sort.Sort(net.ByPubKey(peers))
	pmap := make(map[string]int)
	for i, p := range peers {
		pmap[p.PubKeyHex] = i
	}

	//Find the ID of this node
	nodePub := fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey))
	nodeID := pmap[nodePub]

	logger.WithFields(logrus.Fields{
		"pmap": pmap,
		"id":   nodeID,
	}).Debug("PARTICIPANTS")

	//Instantiate the Store (inmem or badger)
	var store hg.Store
	var needBootstrap bool
	switch storeType {
	case "inmem":
		store = hg.NewInmemStore(pmap, conf.CacheSize)
	case "badger":
		//If the file already exists, load and bootstrap the store using the file
		if _, err := os.Stat(conf.StorePath); err == nil {
			logger.Debug("loading badger store from existing database")
			store, err = hg.LoadBadgerStore(conf.CacheSize, conf.StorePath)
			if err != nil {
				return cli.NewExitError(
					fmt.Sprintf("failed to load BadgerStore from existing file: %s", err),
					1)
			}
			needBootstrap = true
		} else {
			//Otherwise create a new one
			logger.Debug("creating new badger store from fresh database")
			store, err = hg.NewBadgerStore(pmap, conf.CacheSize, conf.StorePath)
			if err != nil {
				return cli.NewExitError(
					fmt.Sprintf("failed to create new BadgerStore: %s", err),
					1)
			}
		}
	default:
		return cli.NewExitError(fmt.Sprintf("invalid store option: %s", storeType), 1)
	}

	trans, err := net.NewTCPTransport(addr,
		nil, maxPool, conf.TCPTimeout, logger)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	var prox proxy.AppProxy
	if noclient {
		prox = aproxy.NewInmemAppProxy(logger)
	} else {
		prox = aproxy.NewSocketAppProxy(clientAddress, proxyAddress,
			conf.TCPTimeout, logger)
	}

	node := node.NewNode(conf, nodeID, key, peers, store, trans, prox)
	if err := node.Init(needBootstrap); err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to initialize node: %s", err),
			1)
	}

	serviceServer := service.NewService(serviceAddress, node, logger)
	go serviceServer.Serve()

	node.Run(true)

	return nil
}

func printVersion(c *cli.Context) error {
	fmt.Println(version.Version)
	return nil
}

//------------------------------------------------------------------------------

func defaultBadgerDir() string {
	dataDir := defaultDataDir()
	if dataDir != "" {
		return filepath.Join(dataDir, "badger_db")
	}
	return ""
}

func defaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".babble")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "BABBLE")
		} else {
			return filepath.Join(home, ".babble")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func logLevel(l string) logrus.Level {
	switch l {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.DebugLevel
	}
}
