package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mosaicnetworks/babble/src/babble"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/node"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/app"
	"github.com/mosaicnetworks/babble/src/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type cliCallback func(*babble.BabbleConfig)

var (
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Directory for the configuration",
		Value: babble.DefaultDataDir(),
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
	StoreFlag = cli.BoolFlag{
		Name:  "store",
		Usage: "Use BadgerDB as local store",
	}
)

func parseConfig(callback cliCallback) {
	app := cli.NewApp()
	app.Name = "babble"
	app.Usage = "hashgraph consensus"
	app.HideVersion = true //there is a special command to print the version
	app.Commands = []cli.Command{
		{
			Name:   "keygen",
			Usage:  "Dump new key pair",
			Action: keygen,
			Flags: []cli.Flag{
				DataDirFlag,
			},
		},
		{
			Name:   "run",
			Usage:  "Run node",
			Action: run(callback),
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
	datadir := c.String(DataDirFlag.Name)

	priv, err := babble.Keygen(datadir)

	if err != nil {
		fmt.Println("Error generating Private key:", err)

		os.Exit(2)
	}

	pemDump, err := crypto.ToPemKey(priv)

	if err != nil {
		fmt.Println("Error generating PemDump:", err)

		os.Exit(2)
	}

	fmt.Println("PublicKey:", pemDump.PublicKey)

	return nil
}

func run(callback cliCallback) func(*cli.Context) error {
	return func(c *cli.Context) error {
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
		store := c.Bool(StoreFlag.Name)

		config := babble.NewDefaultConfig()

		config.Logger.Level = babble.LogLevel(c.String(LogLevelFlag.Name))
		config.BindAddr = addr
		config.ServiceAddr = serviceAddress
		config.Store = store
		config.DataDir = datadir
		config.MaxPool = maxPool

		config.Logger.WithFields(logrus.Fields{
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
			"store":        store,
		}).Debug("RUN")

		config.NodeConfig = node.NewConfig(
			time.Duration(heartbeat)*time.Millisecond,
			time.Duration(tcpTimeout)*time.Millisecond,
			cacheSize,
			syncLimit,
			config.Logger,
		)

		if !noclient {
			p, err := aproxy.NewSocketAppProxy(
				clientAddress,
				proxyAddress,
				config.NodeConfig.TCPTimeout,
				config.Logger,
			)

			if err != nil {
				return err
			}

			config.Proxy = p
		}

		callback(config)

		return nil
	}
}

func printVersion(c *cli.Context) error {
	fmt.Println(version.Version)

	return nil
}
