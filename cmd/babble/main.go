package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	_ "net/http/pprof"

	"github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"

	"github.com/babbleio/babble/crypto"
	"github.com/babbleio/babble/net"
	"github.com/babbleio/babble/node"
	"github.com/babbleio/babble/proxy"
	aproxy "github.com/babbleio/babble/proxy/app"
	"github.com/babbleio/babble/service"
	"github.com/babbleio/babble/version"
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
	}).Debug("RUN")

	conf := node.NewConfig(time.Duration(heartbeat)*time.Millisecond,
		time.Duration(tcpTimeout)*time.Millisecond,
		cacheSize, syncLimit, logger)

	// Create the PEM key
	pemKey := crypto.NewPemKey(datadir)

	// Try a read
	key, err := pemKey.ReadKey()
	if err != nil {
		return err
	}

	// Create the peer store
	store := net.NewJSONPeers(datadir)

	// Try a read
	peers, err := store.Peers()
	if err != nil {
		return err
	}

	// There should be at least two peers
	if len(peers) < 2 {
		return fmt.Errorf("peers.json should define at least two peers")
	}

	trans, err := net.NewTCPTransport(addr,
		nil, maxPool, conf.TCPTimeout, logger)
	if err != nil {
		return err
	}

	var prox proxy.AppProxy
	if noclient {
		prox = aproxy.NewInmemAppProxy(logger)
	} else {
		prox = aproxy.NewSocketAppProxy(clientAddress, proxyAddress,
			conf.TCPTimeout, logger)
	}

	node := node.NewNode(conf, key, peers, trans, prox)
	node.Init()

	serviceServer := service.NewService(serviceAddress, &node, logger)
	go serviceServer.Serve()

	node.Run(true)

	return nil
}

func printVersion(c *cli.Context) error {
	fmt.Println(version.Version)
	return nil
}

//------------------------------------------------------------------------------

func defaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "BABBLE")
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
