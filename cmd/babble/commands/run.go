package commands

import (
	"github.com/mosaicnetworks/babble/src/babble"
	"github.com/mosaicnetworks/babble/src/proxy/dummy"
	aproxy "github.com/mosaicnetworks/babble/src/proxy/socket/app"
	"github.com/mosaicnetworks/monetd/src/configuration"
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
	if !_config.Standalone {
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
	} else {
		p := dummy.NewInmemDummyClient(_config.Babble.Logger())

		_config.Babble.Proxy = p
	}

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

	// Network
	cmd.Flags().StringP("listen", "l", _config.Babble.BindAddr, "Listen IP:Port for babble node")
	cmd.Flags().StringP("advertise", "a", _config.Babble.AdvertiseAddr, "Advertise IP:Port for babble node")
	cmd.Flags().DurationP("timeout", "t", _config.Babble.TCPTimeout, "TCP Timeout")
	cmd.Flags().DurationP("join-timeout", "j", _config.Babble.JoinTimeout, "Join Timeout")
	cmd.Flags().Int("max-pool", _config.Babble.MaxPool, "Connection pool size max")

	// Proxy
	cmd.Flags().Bool("standalone", _config.Standalone, "Do not create a proxy")
	cmd.Flags().StringP("proxy-listen", "p", _config.ProxyAddr, "Listen IP:Port for babble proxy")
	cmd.Flags().StringP("client-connect", "c", _config.ClientAddr, "IP:Port to connect to client")

	// Service
	cmd.Flags().StringP("service-listen", "s", _config.Babble.ServiceAddr, "Listen IP:Port for HTTP service")

	// Store
	cmd.Flags().Bool("store", _config.Babble.Store, "Use badgerDB instead of in-mem DB")
	cmd.Flags().String("db", _config.Babble.DatabaseDir, "Dabatabase directory")
	cmd.Flags().Bool("bootstrap", _config.Babble.Bootstrap, "Load from database")
	cmd.Flags().Int("cache-size", _config.Babble.CacheSize, "Number of items in LRU caches")

	// Node configuration
	cmd.Flags().Duration("heartbeat", _config.Babble.HeartbeatTimeout, "Time between gossips")
	cmd.Flags().Int("sync-limit", _config.Babble.SyncLimit, "Max number of events for sync")
	cmd.Flags().Bool("fast-sync", _config.Babble.EnableFastSync, "Enable FastSync")
}

func loadConfig(cmd *cobra.Command, args []string) error {

	err := bindFlagsLoadViper(cmd)
	if err != nil {
		return err
	}

	// If --datadir was explicitely set, but not --db, this will update the
	// default database dir to be inside the new datadir
	_config.Babble.SetDataDir(_config.Babble.DataDir)

	logFields := logrus.Fields{
		"babble.DataDir":          _config.Babble.DataDir,
		"babble.BindAddr":         _config.Babble.BindAddr,
		"babble.AdvertiseAddr":    _config.Babble.AdvertiseAddr,
		"babble.ServiceAddr":      _config.Babble.ServiceAddr,
		"babble.MaxPool":          _config.Babble.MaxPool,
		"babble.Store":            _config.Babble.Store,
		"babble.LoadPeers":        _config.Babble.LoadPeers,
		"babble.LogLevel":         _config.Babble.LogLevel,
		"babble.Moniker":          _config.Babble.Moniker,
		"babble.HeartbeatTimeout": _config.Babble.HeartbeatTimeout,
		"babble.TCPTimeout":       _config.Babble.TCPTimeout,
		"babble.JoinTimeout":      _config.Babble.JoinTimeout,
		"babble.CacheSize":        _config.Babble.CacheSize,
		"babble.SyncLimit":        _config.Babble.SyncLimit,
		"babble.EnableFastSync":   _config.Babble.EnableFastSync,
		"ProxyAddr":               _config.ProxyAddr,
		"ClientAddr":              _config.ClientAddr,
		"Standalone":              _config.Standalone,
	}

	if _config.Babble.Store {
		logFields["babble.DatabaseDir"] = _config.Babble.DatabaseDir
		logFields["babble.Bootstrap"] = _config.Babble.Bootstrap
	}

	_config.Babble.Logger().WithFields(logFields).Debug("RUN")

	return nil
}

// Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command) error {
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
		_config.Babble.Logger().Debugf("No config file found in: %s", _config.Babble.DataDir)
	} else {
		return err
	}

	// second unmarshal to read from config file
	return viper.Unmarshal(configuration.Global)
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
