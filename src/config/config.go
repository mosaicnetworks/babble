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
	webrtc "github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Default filenames.
const (
	// DefaultKeyfile is the default name of the file containing the validator's
	// private key
	DefaultKeyfile = "priv_key"

	// DefaultBadgerFile is the default name of the folder containing the Badger
	// database
	DefaultBadgerFile = "badger_db"

	// DefaultCertFile is the default name of the file containing the TLS
	// certificate for connecting to the signaling server.
	DefaultCertFile = "cert.pem"
)

// Default configuration values.
const (
	DefaultLogLevel             = "debug"
	DefaultBindAddr             = "127.0.0.1:1337"
	DefaultServiceAddr          = "127.0.0.1:8000"
	DefaultHeartbeatTimeout     = 10 * time.Millisecond
	DefaultSlowHeartbeatTimeout = 1000 * time.Millisecond
	DefaultTCPTimeout           = 1000 * time.Millisecond
	DefaultJoinTimeout          = 10000 * time.Millisecond
	DefaultCacheSize            = 10000
	DefaultSyncLimit            = 1000
	DefaultMaxPool              = 2
	DefaultStore                = false
	DefaultMaintenanceMode      = false
	DefaultSuspendLimit         = 100
	DefaultWebRTC               = false
	DefaultSignalAddr           = "127.0.0.1:2443"
	DefaultSignalRealm          = "main"
	DefaultSignalSkipVerify     = false
	DefaultICEAddress           = "stun:stun.l.google.com:19302"
	DefaultICEUsername          = ""
	DefaultICEPassword          = ""
)

// Config contains all the configuration properties of a Babble node.
type Config struct {
	// DataDir is the top-level directory containing Babble configuration and
	// data
	DataDir string `mapstructure:"datadir"`

	// LogLevel determines the chattiness of the log output.
	LogLevel string `mapstructure:"log"`

	// BindAddr is the local address:port where this node gossips with other
	// nodes. in some cases, there may be a routable address that cannot be
	// bound. Use AdvertiseAddr to advertise a different address to support
	// this. If this address is not routable, the node will be in a constant
	// flapping state as other nodes will treat the non-routability as a
	// failure.
	BindAddr string `mapstructure:"listen"`

	// AdvertiseAddr is used to change the address that we advertise to other
	// nodes.
	AdvertiseAddr string `mapstructure:"advertise"`

	// NoService disables the HTTP API service.
	NoService bool `mapstructure:"no-service"`

	// ServiceAddr is the address:port of the optional HTTP service. If not
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

	// TCPTimeout is the timeout of gossip RPC connections. It also applies to
	// WebRTC connections.
	TCPTimeout time.Duration `mapstructure:"timeout"`

	// JoinTimeout is the timeout of Join Requests
	JoinTimeout time.Duration `mapstructure:"join_timeout"`

	// SyncLimit defines the max number of hashgraph events to include in a
	// SyncResponse or EagerSyncRequest
	SyncLimit int `mapstructure:"sync-limit"`

	// EnableFastSync enables the FastSync protocol.
	EnableFastSync bool `mapstructure:"fast-sync"`

	// Store activates persistant storage.
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

	// SuspendLimit is the multiplier that is dynamically applied to the number
	// of validators to determine the limit of undetermined events (events which
	// haven't reached consensus) that will cause the node to become suspended.
	// For example, if there are 4 validators and SuspendLimit=100, then the
	// node will suspend itself after registering 400 undetermined events.
	SuspendLimit int `mapstructure:"suspend-limit"`

	// Moniker defines the friendly name of this node
	Moniker string `mapstructure:"moniker"`

	// WebRTC determines whether to use a WebRTC transport. WebRTC uses a very
	// different protocol stack than TCP/IP and enables peers to connect
	// directly even with multiple layers of NAT between them, such as in
	// cellular networks. WebRTC relies on a signalling server who's address is
	// specified by SignalAddr. When WebRTC is enabled, BindAddr and
	// AdvertiseAddr are ignored.
	WebRTC bool `mapstructure:"webrtc"`

	// SignalAddr is the IP:PORT of the WebRTC signaling server. It is ignored
	// when WebRTC is not enabled. The connection is over secured web-sockets,
	// wss, and it possible to include a self-signed certificated in a file
	// called cert.pem in the datadir. If no self-signed certificate is found,
	// the server's certifacate signing authority better be trusted.
	SignalAddr string `mapstructure:"signal-addr"`

	// SignalRealm is an administrative domain within the WebRTC signaling
	// server. WebRTC signaling messages are only routed within a Realm.
	SignalRealm string `mapstructure:"signal-realm"`

	// SignalSkipVerify controls whether the signal client verifies the server's
	// certificate chain and host name. If SignalSkipVerify is true, TLS accepts
	// any certificate presented by the server and any host name in that
	// certificate. In this mode, TLS is susceptible to man-in-the-middle
	// attacks. This should be used only for testing.
	SignalSkipVerify bool `mapstructure:"signal-skip-verify"`

	// ICE address is the URI of a server providing services for ICE, such as
	// STUN and TURN. The server should support password-based authentication,
	// as Babble will try to connect with the username and password provided in
	// ICEUsername and ICEPassword below. Username adn password can also be
	// empty if the ICE server does not use authentication.
	// https://developer.mozilla.org/en-US/docs/Web/API/RTCIceServer/urls
	ICEAddress string `mapstructure:"ice-addr"`

	// ICEUsername is the username that will be used to authenticate with the
	// ICE server defined in ICEAddress.
	ICEUsername string `mapstructure:"ice-username"`

	// ICEPassword is the password that will be used to authenticate with the
	// ICE server defined in ICEAddress.
	ICEPassword string `mapstructure:"ice-password"`

	// Proxy is the application proxy that enables Babble to communicate with
	// the application.
	Proxy proxy.AppProxy

	// Key is the private key of the validator.
	Key *ecdsa.PrivateKey

	logger *logrus.Logger
}

// NewDefaultConfig returns a config object with default values. All the default
// configuration values are set, even if they cancel eachother out. For example,
// When WebRTC = false, all the Signal options are ignored. Likewise, when
// WebRTC = true, BindAddr and ServiceAddr are not used.
func NewDefaultConfig() *Config {
	config := &Config{
		DataDir:              DefaultDataDir(),
		LogLevel:             DefaultLogLevel,
		BindAddr:             DefaultBindAddr,
		ServiceAddr:          DefaultServiceAddr,
		HeartbeatTimeout:     DefaultHeartbeatTimeout,
		SlowHeartbeatTimeout: DefaultSlowHeartbeatTimeout,
		TCPTimeout:           DefaultTCPTimeout,
		JoinTimeout:          DefaultJoinTimeout,
		CacheSize:            DefaultCacheSize,
		SyncLimit:            DefaultSyncLimit,
		MaxPool:              DefaultMaxPool,
		Store:                DefaultStore,
		MaintenanceMode:      DefaultMaintenanceMode,
		DatabaseDir:          DefaultDatabaseDir(),
		SuspendLimit:         DefaultSuspendLimit,
		WebRTC:               DefaultWebRTC,
		SignalAddr:           DefaultSignalAddr,
		SignalRealm:          DefaultSignalRealm,
		SignalSkipVerify:     DefaultSignalSkipVerify,
		ICEAddress:           DefaultICEAddress,
		ICEUsername:          DefaultICEUsername,
		ICEPassword:          DefaultICEPassword,
	}

	return config
}

// NewTestConfig returns a config object with default values and a special
// logger for debugging tests.
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

// CertFile returns the full path of the file containing the signal-server TLS
// certificate.
func (c *Config) CertFile() string {
	return filepath.Join(c.DataDir, DefaultCertFile)
}

// ICEServers returns a list of ICE servers used by the WebRTCStreamLayer to
// connect to peers. The list contains a single item which is based on the
// configuration passed through the config object. This configuration is limited
// to a single server, with password-based authentication.
func (c *Config) ICEServers() []webrtc.ICEServer {
	return []webrtc.ICEServer{
		{
			URLs:           []string{c.ICEAddress},
			Username:       c.ICEUsername,
			Credential:     c.ICEPassword,
			CredentialType: webrtc.ICECredentialTypePassword,
		},
	}
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

// DefaultICEServers returns the default ICE configuration with one URL pointing
// to a public Google STUN server.
func DefaultICEServers() []webrtc.ICEServer {
	return []webrtc.ICEServer{
		{
			URLs: []string{DefaultICEAddress},
		},
	}
}
