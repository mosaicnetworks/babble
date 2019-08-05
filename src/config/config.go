package config

import (
	"crypto/ecdsa"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

	// BindAddr is the address:port where this node gossips with other nodes.
	BindAddr string `mapstructure:"listen"`

	// ServiceAddr is the address:port that serves the user-facing API.
	ServiceAddr string `mapstructure:"service-listen"`

	// HeartbeatTimeout is the frequency of the gossip timer.
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat"`

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

	// CacheSize is the max number of items in in-memory caches.
	CacheSize int `mapstructure:"cache-size"`

	// Bootstrap determines whether or not to load Babble from an existing
	// database file.
	Bootstrap bool `mapstructure:"bootstrap"`

	// Moniker defines the friendly name of this node
	Moniker string `mapstructure:"moniker"`

	// LoadPeers determines whether or not to attempt loading the peer-set from
	// a local json file.
	LoadPeers bool

	// Proxy is the application proxy that enables Babble to communicate with
	// application.
	Proxy proxy.AppProxy

	// Key is the private key of the validator.
	Key *ecdsa.PrivateKey

	logger *logrus.Logger
}

// NewDefaultConfig ...
func NewDefaultConfig() *Config {

	config := &Config{
		DataDir:          DefaultDataDir(),
		LogLevel:         "debug",
		BindAddr:         "127.0.0.1:1337",
		ServiceAddr:      "127.0.0.1:8000",
		HeartbeatTimeout: 10 * time.Millisecond,
		TCPTimeout:       1000 * time.Millisecond,
		JoinTimeout:      10000 * time.Millisecond,
		CacheSize:        5000,
		SyncLimit:        1000,
		MaxPool:          2,
		Store:            false,
		LoadPeers:        true,
	}

	return config
}

// NewTestConfig returns a Preset Test Configuration
func NewTestConfig(t testing.TB) *Config {
	config := NewDefaultConfig()

	config.logger = logrus.New()
	config.logger.Level = LogLevel(config.LogLevel)
	config.logger.Formatter = new(prefixed.TextFormatter)
	config.logger.SetReportCaller(true)

	return config
}

// BadgerDir returs the full path of the folder containing the Babdger database.
func (c *Config) BadgerDir() string {
	return filepath.Join(c.DataDir, DefaultBadgerFile)
}

// Keyfile returns the full path of the file containing the private key.
func (c *Config) Keyfile() string {
	return filepath.Join(c.DataDir, DefaultKeyfile)
}

// Logger returns the logrus Entry
func (c *Config) Logger() *logrus.Entry {
	if c.logger == nil {
		c.logger = logrus.New()
		c.logger.Level = LogLevel(c.LogLevel)
		c.logger.Formatter = new(prefixed.TextFormatter)
	}
	return c.logger.WithField("prefix", "BABBLE")
}

// DefaultDataDir ...
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := HomeDir()
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

// HomeDir ...
func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

// LogLevel ...
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
