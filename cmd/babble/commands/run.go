package commands

import (
	"github.com/mosaicnetworks/babble/src/babble"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/app"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//AddRunFlags adds flags to the Run command
func AddRunFlags(cmd *cobra.Command) {

	cmd.Flags().String("datadir", config.Babble.DataDir, "Top-level directory for configuration and data")
	cmd.Flags().String("log", config.Babble.LogLevel, "debug, info, warn, error, fatal, panic")

	// Network
	cmd.Flags().StringP("listen", "l", config.Babble.BindAddr, "Listen IP:Port for babble node")
	cmd.Flags().DurationP("timeout", "t", config.Babble.NodeConfig.TCPTimeout, "TCP Timeout")
	cmd.Flags().Int("max-pool", config.Babble.MaxPool, "Connection pool size max")

	// Proxy
	cmd.Flags().Bool("inapp", config.Inapp, "Use an in-app proxy")
	cmd.Flags().StringP("proxy-listen", "p", config.ProxyAddr, "Listen IP:Port for babble proxy")
	cmd.Flags().StringP("client-connect", "c", config.ClientAddr, "IP:Port to connect to client")

	// Service
	cmd.Flags().StringP("service-listen", "s", config.Babble.ServiceAddr, "Listen IP:Port for HTTP service")

	// Store
	cmd.Flags().Bool("store", config.Babble.Store, "Use badgerDB instead of in-mem DB")
	cmd.Flags().Int("cache-size", config.Babble.NodeConfig.CacheSize, "Number of items in LRU caches")

	// Node configuration
	cmd.Flags().Duration("heartbeat", config.Babble.NodeConfig.HeartbeatTimeout, "Time between gossips")
	cmd.Flags().Int("sync-limit", config.Babble.NodeConfig.SyncLimit, "Max number of events for sync")
}

//NewRunCmd returns the command that starts a Babble node
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run node",
		PreRunE: logConfig,
		RunE:    runBabble,
	}

	AddRunFlags(cmd)

	return cmd
}

func logConfig(cmd *cobra.Command, args []string) error {
	config.Babble.Logger.Level = babble.LogLevel(config.Babble.LogLevel)
	config.Babble.NodeConfig.Logger = config.Babble.Logger

	config.Babble.Logger.WithFields(logrus.Fields{
		"proxy-listen":   config.ProxyAddr,
		"client-connect": config.ClientAddr,
		"inapp":          config.Inapp,

		"babble.datadir":        config.Babble.DataDir,
		"babble.bindaddr":       config.Babble.BindAddr,
		"babble.service-listen": config.Babble.ServiceAddr,
		"babble.maxpool":        config.Babble.MaxPool,
		"babble.store":          config.Babble.Store,
		"babble.loadpeers":      config.Babble.LoadPeers,
		"babble.log":            config.Babble.LogLevel,

		"babble.node.heartbeat":  config.Babble.NodeConfig.HeartbeatTimeout,
		"babble.node.tcptimeout": config.Babble.NodeConfig.TCPTimeout,
		"babble.node.cachesize":  config.Babble.NodeConfig.CacheSize,
		"babble.node.synclimit":  config.Babble.NodeConfig.SyncLimit,
	}).Debug("RUN")

	return nil
}

func runBabble(cmd *cobra.Command, args []string) error {
	if !config.Inapp {
		p, err := aproxy.NewSocketAppProxy(
			config.ClientAddr,
			config.ProxyAddr,
			config.Babble.NodeConfig.HeartbeatTimeout,
			config.Babble.Logger,
		)

		if err != nil {
			config.Babble.Logger.Error("Cannot initialize socket AppProxy:", err)
			return nil
		}

		config.Babble.Proxy = p
	}

	engine := babble.NewBabble(&config.Babble)

	if err := engine.Init(); err != nil {
		config.Babble.Logger.Error("Cannot initialize engine:", err)
		return nil
	}

	engine.Run()

	return nil
}
