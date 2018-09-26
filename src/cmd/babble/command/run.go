package command

import (
	"fmt"
	"os"

	"github.com/mosaicnetworks/babble/src/babble"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/app"
	"github.com/mosaicnetworks/babble/src/service"
	vers "github.com/mosaicnetworks/babble/src/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CliConfig struct {
	Babble      babble.BabbleConfig `mapstructure:",squash"`
	ProxyAddr   string              `mapstructure:"proxy-listen"`
	ClientAddr  string              `mapstructure:"client-connect"`
	ServiceAddr string              `mapstructure:"service-listen"`
	Inapp       bool                `mapstructure:"inapp"`
}

func NewDefaultCliConfig() *CliConfig {
	return &CliConfig{
		Babble:      *babble.NewDefaultConfig(),
		ProxyAddr:   "127.0.0.1:1338",
		ClientAddr:  "127.0.0.1:1339",
		ServiceAddr: "127.0.0.1:8000",
		Inapp:       false,
	}
}

var (
	config  *CliConfig
	datadir *string
	version *bool
)

func init() {
	config = NewDefaultCliConfig()

	cobra.OnInitialize(initConfig)

	// Base datadir
	datadir = rootCmd.PersistentFlags().StringP("datadir", "d", config.Babble.DataDir, "Base configuration directory")

	// Listen and connect addresses
	rootCmd.PersistentFlags().StringP("listen", "l", config.Babble.BindAddr, "Listen IP:Port for babble node")
	rootCmd.PersistentFlags().StringP("proxy-listen", "p", config.ProxyAddr, "Listen IP:Port for the babble proxy")
	rootCmd.PersistentFlags().StringP("client-connect", "c", config.ClientAddr, "IP:Port of the app client proxy to connect to")
	rootCmd.PersistentFlags().StringP("service-listen", "s", config.ServiceAddr, "HTTP service listen to IP:Port")

	// Various
	rootCmd.PersistentFlags().Bool("inapp", config.Inapp, "Use an in-app proxy")
	rootCmd.PersistentFlags().Bool("store", config.Babble.Store, "Use badgerDB instead of in-mem DB")
	rootCmd.PersistentFlags().String("log", config.Babble.LogLevel, "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().Int("max-pool", config.Babble.MaxPool, "Connection pool size max")

	// Node configuration
	rootCmd.PersistentFlags().Duration("heartbeat", config.Babble.NodeConfig.HeartbeatTimeout, "Time between gossips")
	rootCmd.PersistentFlags().DurationP("timeout", "t", config.Babble.NodeConfig.TCPTimeout, "TCP Timeout")
	rootCmd.PersistentFlags().Int("cache-size", config.Babble.NodeConfig.CacheSize, "Number of items in LRU caches")
	rootCmd.PersistentFlags().Int("sync-limit", config.Babble.NodeConfig.SyncLimit, "Max amount of events for sync")

	// Version
	version = rootCmd.PersistentFlags().BoolP("version", "v", false, "Show version and exit")

}

func initConfig() {
	viper.AddConfigPath(*datadir)
	viper.SetConfigName("babble")

	viper.BindPFlags(rootCmd.PersistentFlags())

	if err := viper.ReadInConfig(); err != nil {
		config.Babble.Logger.Warn(err, ". Taking cli or default.")
	}

	err := viper.Unmarshal(config)
	if err != nil {
		config.Babble.Logger.Warn(err, ". Taking cli or default.")
	}
}

var rootCmd = &cobra.Command{
	Use:   "babble",
	Short: "Babble Hashgraph consensus system",
	Long:  "Babble Hashgraph consensus system",
	Run: func(cmd *cobra.Command, args []string) {
		if *version {
			fmt.Println(vers.Version)

			return
		}

		config.Babble.Logger.Level = babble.LogLevel(config.Babble.LogLevel)
		config.Babble.NodeConfig.Logger = config.Babble.Logger

		config.Babble.Logger.WithFields(logrus.Fields{
			"proxy-listen":   config.ProxyAddr,
			"client-connect": config.ClientAddr,
			"service-listen": config.ServiceAddr,
			"inapp":          config.Inapp,

			"babble.datadir":   config.Babble.DataDir,
			"babble.bindaddr":  config.Babble.BindAddr,
			"babble.maxpool":   config.Babble.MaxPool,
			"babble.store":     config.Babble.Store,
			"babble.loadpeers": config.Babble.LoadPeers,
			"babble.log":       config.Babble.LogLevel,

			"babble.node.heartbeat":  config.Babble.NodeConfig.HeartbeatTimeout,
			"babble.node.tcptimeout": config.Babble.NodeConfig.TCPTimeout,
			"babble.node.cachesize":  config.Babble.NodeConfig.CacheSize,
			"babble.node.synclimit":  config.Babble.NodeConfig.SyncLimit,
		}).Debug("RUN")

		if !config.Inapp {
			p, err := aproxy.NewSocketAppProxy(
				config.ClientAddr,
				config.ProxyAddr,
				config.Babble.NodeConfig.HeartbeatTimeout,
				config.Babble.Logger,
			)

			if err != nil {
				config.Babble.Logger.Error("Cannot initialize socket AppProxy:", err)

				return
			}

			config.Babble.Proxy = p
		}

		engine := babble.NewBabble(&config.Babble)

		if err := engine.Init(); err != nil {
			config.Babble.Logger.Error("Cannot initialize engine:", err)

			return
		}

		serviceServer := service.NewService(config.ServiceAddr, engine.Node, config.Babble.Logger)

		go serviceServer.Serve()

		engine.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)

		os.Exit(1)
	}
}
