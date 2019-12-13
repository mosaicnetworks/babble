package commands

import (
	"path/filepath"

	"github.com/mosaicnetworks/babble/src/babble"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/socket/app"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//NewRunCmd returns the command that starts a Babble node
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run node",
		PreRunE: bindFlagsLoadViper,
		RunE:    runBabble,
	}
	AddRunFlags(cmd)
	return cmd
}

/*******************************************************************************
* RUN
*******************************************************************************/

func runBabble(cmd *cobra.Command, args []string) error {

	_config.Babble.Logger().WithFields(logrus.Fields{
		"ProxyAddr":  _config.ProxyAddr,
		"ClientAddr": _config.ClientAddr,
	}).Debug("Config Proxy")

	p, err := aproxy.NewSocketAppProxy(
		_config.ClientAddr,
		_config.ProxyAddr,
		_config.Babble.HeartbeatTimeout,
		_config.Babble.Logger(),
	)

	if err != nil {
		_config.Babble.Logger().Error("Cannot initialize socket AppProxy:", err)
		return err
	}

	_config.Babble.Proxy = p

	engine := babble.NewBabble(&_config.Babble)

	if err := engine.Init(); err != nil {
		_config.Babble.Logger().Error("Cannot initialize engine:", err)
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

	cmd.Flags().String("datadir", _config.Babble.DataDir, "Top-level directory for configuration and data")
	cmd.Flags().String("log", _config.Babble.LogLevel, "debug, info, warn, error, fatal, panic")
	cmd.Flags().String("moniker", _config.Babble.Moniker, "Optional name")
	cmd.Flags().BoolP("maintenance-mode", "R", _config.Babble.MaintenanceMode, "Start Babble in a suspended (non-gossipping) state")

	// Network
	cmd.Flags().StringP("listen", "l", _config.Babble.BindAddr, "Listen IP:Port for babble node")
	cmd.Flags().StringP("advertise", "a", _config.Babble.AdvertiseAddr, "Advertise IP:Port for babble node")
	cmd.Flags().DurationP("timeout", "t", _config.Babble.TCPTimeout, "TCP Timeout")
	cmd.Flags().DurationP("join-timeout", "j", _config.Babble.JoinTimeout, "Join Timeout")
	cmd.Flags().Int("max-pool", _config.Babble.MaxPool, "Connection pool size max")

	// Proxy
	cmd.Flags().StringP("proxy-listen", "p", _config.ProxyAddr, "Listen IP:Port for babble proxy")
	cmd.Flags().StringP("client-connect", "c", _config.ClientAddr, "IP:Port to connect to client")

	// Service
	cmd.Flags().Bool("no-service", _config.Babble.NoService, "Disable HTTP service")
	cmd.Flags().StringP("service-listen", "s", _config.Babble.ServiceAddr, "Listen IP:Port for HTTP service")

	// Store
	cmd.Flags().Bool("store", _config.Babble.Store, "Use badgerDB instead of in-mem DB")
	cmd.Flags().String("db", _config.Babble.DatabaseDir, "Dabatabase directory")
	cmd.Flags().Bool("bootstrap", _config.Babble.Bootstrap, "Load from database")
	cmd.Flags().Int("cache-size", _config.Babble.CacheSize, "Number of items in LRU caches")

	// Node configuration
	cmd.Flags().Duration("heartbeat", _config.Babble.HeartbeatTimeout, "Timer frequency when there is something to gossip about")
	cmd.Flags().Duration("slow-heartbeat", _config.Babble.SlowHeartbeatTimeout, "Timer frequency when there is nothing to gossip about")
	cmd.Flags().Int("sync-limit", _config.Babble.SyncLimit, "Max number of events for sync")
	cmd.Flags().Bool("fast-sync", _config.Babble.EnableFastSync, "Enable FastSync")
	cmd.Flags().Int("suspend-limit", _config.Babble.SuspendLimit, "Limit of undetermined events before entering suspended state")
}

// Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command, args []string) error {
	// Register flags with viper. Include flags from this command and all other
	// persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// first unmarshal to read from CLI flags
	if err := viper.Unmarshal(_config); err != nil {
		return err
	}

	// look for config file in [datadir]/babble.toml (.json, .yaml also work)
	viper.SetConfigName("babble")               // name of config file (without extension)
	viper.AddConfigPath(_config.Babble.DataDir) // search root directory

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_config.Babble.Logger().Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		_config.Babble.Logger().Debugf("No config file found in: %s", filepath.Join(_config.Babble.DataDir, "babble.toml"))
	} else {
		return err
	}

	// second unmarshal to read from config file
	return viper.Unmarshal(_config)
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
