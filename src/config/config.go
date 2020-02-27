package config

import (
	"crypto/ecdsa"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const (
	// DefaultKeyfile defines the default name of the file containing the
	// validator's private key
	DefaultKeyfile = "priv_key"

	// DefaultBadgerFile defines the default name of the folder containing the
	// Badger database
	DefaultBadgerFile = "badger_db"
)

// Config contains all the configuration properties of a Babble node.
type Config struct {
	// DataDir is the top-level directory containing Babble configuration and
	// data
	DataDir string `mapstructure:"datadir"`

	// LogLevel determines the chattiness of the log output.
	LogLevel string `mapstructure:"log"`

	// BindAddr is the local address:port where this node gossips with other
	// nodes. By default, this is "0.0.0.0", meaning Babble will bind to all
	// addresses on the local machine. However, in some cases, there may be a
	// routable address that cannot be bound. Use AdvertiseAddr to enable
	// gossiping a different address to support this. If this address is not
	// routable, the node will be in a constant flapping state as other nodes
	// will treat the non-routability as a failure
	BindAddr string `mapstructure:"listen"`

	// AdvertiseAddr is used to change the address that we advertise to other
	// nodes in the cluster
	AdvertiseAddr string `mapstructure:"advertise"`

	// NoService disables the HTTP API service.
	NoService bool `mapstructure:"no-service"`

	// ServiceAddr is the address:port that serves the user-facing API. If not
	// specified, and "no-service" is not set, the API handlers are registered
	// with the DefaultServerMux of the http package. It is possible that
	// another server in the same process is simultaneously using the
	// DefaultServerMux. In which case, the handlers will be accessible from
	// both servers. This is usefull when Babble is used in-memory and expected
	// to use the same endpoint (address:port) as the application's API.
	ServiceAddr string `mapstructure:"service-listen"`

	// HeartbeatTimeout is the frequency of the gossip timer when the node has
	// something to gossip about.
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat"`

	// SlowHeartbeatTimeout is the frequency of the gossip timer when the node
	// has nothing to gossip about.
	SlowHeartbeatTimeout time.Duration `mapstructure:"slow-heartbeat"`

	// MaxPool controls how many connections are pooled per target in the gossip
	// routines.
	MaxPool int `mapstructure:"max-pool"`

	// TCPTimeout is the timeout of gossip TCP connections.
	TCPTimeout time.Duration `mapstructure:"timeout"`

	// JoinTimeout is the timeout of Join Requests
	JoinTimeout time.Duration `mapstructure:"join_timeout"`

	// SyncLimit defines the max number of hashgraph events to include in a
	// SyncResponse or EagerSyncRequest
	SyncLimit int `mapstructure:"sync-limit"`

	// EnableFastSync determines whether or not to enable the FastSync protocol.
	EnableFastSync bool `mapstructure:"fast-sync"`

	// Store is a flag that determines whether or not to use persistant storage.
	Store bool `mapstructure:"store"`

	// DatabaseDir is the directory containing database files.
	DatabaseDir string `mapstructure:"db"`

	// CacheSize is the max number of items in in-memory caches.
	CacheSize int `mapstructure:"cache-size"`

	// Bootstrap determines whether or not to load Babble from an existing
	// database file. Forces Store, ie. bootstrap only works with a persistant
	// database store.
	Bootstrap bool `mapstructure:"bootstrap"`

	// MaintenanceMode when set to true causes Babble to initialise in a
	// suspended state. I.e. it does not start gossipping. Forces Bootstrap,
	// which itself forces Store. I.e. MaintenanceMode only works if the node is
	// bootstrapped from an existing database.
	MaintenanceMode bool `mapstructure:"maintenance-mode"`

	// SuspendLimit is the multiplyer that is dynamically applied to the number
	// of validators to determine the limit of undertermined events (events
	// which haven't reached consensus) that will cause the node to become
	// suspended. For example, if there are 4 validators and SuspendLimit=100,
	// then the node will suspend itself after registering 400 undetermined
	// events.
	SuspendLimit int `mapstructure:"suspend-limit"`

	// Moniker defines the friendly name of this node
	Moniker string `mapstructure:"moniker"`

	// LoadPeers determines whether or not to attempt loading the peer-set from
	// a local json file.
	LoadPeers bool `mapstructure:"loadpeers"`

	// XXX
	WebRTC bool `mapstructure:"webrtc"`

	// XXX
	SignalAddr string `mapstructure:"signal-addr"`

	// Proxy is the application proxy that enables Babble to communicate with
	// the application.
	Proxy proxy.AppProxy

	// Key is the private key of the validator.
	Key *ecdsa.PrivateKey

	logger *logrus.Logger
}

// NewDefaultConfig returns the a config object with default values.
func NewDefaultConfig() *Config {

	config := &Config{
		DataDir:              DefaultDataDir(),
		LogLevel:             "debug",
		BindAddr:             "127.0.0.1:1337",
		ServiceAddr:          "127.0.0.1:8000",
		HeartbeatTimeout:     10 * time.Millisecond,
		SlowHeartbeatTimeout: 1000 * time.Millisecond,
		TCPTimeout:           1000 * time.Millisecond,
		JoinTimeout:          10000 * time.Millisecond,
		CacheSize:            5000,
		SyncLimit:            1000,
		MaxPool:              2,
		Store:                false,
		MaintenanceMode:      false,
		DatabaseDir:          DefaultDatabaseDir(),
		LoadPeers:            true,
		SuspendLimit:         100,
	}

	return config
}

// NewTestConfig returns a config object with default values and a special
// logger. the logger forces formatting and colors even when there is no tty
// attached, which makes for more readable logs. The logger also provides info
// about the calling function.
func NewTestConfig(t testing.TB, level logrus.Level) *Config {
	config := NewDefaultConfig()
	config.logger = common.NewTestLogger(t, level)
	return config
}

// SetDataDir sets the top-level Babble directory, and updates the database
// directory if it is currently set to the default value. If the database
// directory is not currently the default, it means the user has explicitely set
// it to something else, so avoid changing it again here.
func (c *Config) SetDataDir(dataDir string) {
	c.DataDir = dataDir
	if c.DatabaseDir == DefaultDatabaseDir() {
		c.DatabaseDir = filepath.Join(dataDir, DefaultBadgerFile)
	}
}

// Keyfile returns the full path of the file containing the private key.
func (c *Config) Keyfile() string {
	return filepath.Join(c.DataDir, DefaultKeyfile)
}

// Logger returns a formatted logrus Entry, with prefix set to "babble".
func (c *Config) Logger() *logrus.Entry {
	if c.logger == nil {
		c.logger = logrus.New()
		c.logger.Level = LogLevel(c.LogLevel)
		c.logger.Formatter = new(prefixed.TextFormatter)
	}
	return c.logger.WithField("prefix", "babble")
}

// DefaultDatabaseDir returns the default path for the badger database files.
func DefaultDatabaseDir() string {
	return filepath.Join(DefaultDataDir(), DefaultBadgerFile)
}

// DefaultDataDir return the default directory name for top-level Babble config
// based on the underlying OS, attempting to respect conventions.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := HomeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".Babble")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Babble")
		} else {
			return filepath.Join(home, ".babble")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

// HomeDir returns the user's home directory.
func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

// LogLevel parses a string into a Logrus log level.
func LogLevel(l string) logrus.Level {
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
