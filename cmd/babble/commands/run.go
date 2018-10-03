package commands

import (
	"github.com/mosaicnetworks/babble/src/babble"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/app"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//NewRunCmd returns the command that starts a Babble node
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run node",
		PreRunE: loadConfig,
		RunE:    runBabble,
	}
	AddRunFlags(cmd)
	return cmd
}

/*******************************************************************************
* RUN
*******************************************************************************/

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
			return err
		}

		config.Babble.Proxy = p
	}

	engine := babble.NewBabble(&config.Babble)

	if err := engine.Init(); err != nil {
		config.Babble.Logger.Error("Cannot initialize engine:", err)
		return err
	}

	engine.Run()

	return nil
}

/*******************************************************************************
* CONFIG
*******************************************************************************/

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

func loadConfig(cmd *cobra.Command, args []string) error {

	err := bindFlagsLoadViper(cmd)
	if err != nil {
		return err
	}

	config, err = parseConfig()
	if err != nil {
		return err
	}

	config.Babble.Logger.Level = babble.LogLevel(config.Babble.LogLevel)
	config.Babble.NodeConfig.Logger = config.Babble.Logger

	config.Babble.Logger.WithFields(logrus.Fields{
		"babble.DataDir":               config.Babble.DataDir,
		"babble.BindAddr":              config.Babble.BindAddr,
		"babble.ServiceAddr":           config.Babble.ServiceAddr,
		"babble.MaxPool":               config.Babble.MaxPool,
		"babble.Store":                 config.Babble.Store,
		"babble.LoadPeers":             config.Babble.LoadPeers,
		"babble.LogLevel":              config.Babble.LogLevel,
		"babble.Node.HeartbeatTimeout": config.Babble.NodeConfig.HeartbeatTimeout,
		"babble.Node.TCPTimeout":       config.Babble.NodeConfig.TCPTimeout,
		"babble.node.CacheSize":        config.Babble.NodeConfig.CacheSize,
		"babble.node.SyncLimit":        config.Babble.NodeConfig.SyncLimit,
		"ProxyAddr":                    config.ProxyAddr,
		"ClientAddr":                   config.ClientAddr,
		"Inapp":                        config.Inapp,
	}).Debug("RUN")

	return nil
}

//Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command) error {
	// cmd.Flags() includes flags from this command and all persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	viper.SetConfigName("babble")              // name of config file (without extension)
	viper.AddConfigPath(config.Babble.DataDir) // search root directory
	// viper.AddConfigPath(filepath.Join(config.Babble.DataDir, "babble")) // search root directory /config

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		config.Babble.Logger.Debugf("Using config file: ", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		config.Babble.Logger.Debugf("No config file found in:", config.Babble.DataDir)
	} else {
		return err
	}

	return nil
}

//Retrieve the default environment configuration.
func parseConfig() (*CLIConfig, error) {
	conf := NewDefaultCLIConfig()
	err := viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
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
