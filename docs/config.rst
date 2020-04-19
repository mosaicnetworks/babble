.. _config:

Configuration
=============

Babble consumes the following configuration object:

.. code:: go

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
    
    	// TCPTimeout is the timeout of gossip TCP connections.
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
    
    	// SuspendLimit is the multiplyer that is dynamically applied to the number
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
    
    	// Proxy is the application proxy that enables Babble to communicate with
    	// the application.
    	Proxy proxy.AppProxy
    
    	// Key is the private key of the validator.
    	Key *ecdsa.PrivateKey
    
    	logger *logrus.Logger
    }

Data Directory
--------------

Babble reads additional configuration from the directory specified by the 
``DataDir`` option which defaults to ``~/.babble`` on Linux. This directory 
should contain the following files:

- ``priv_key``    : The private key of the validator runnning the node. This is
  optional if the Key field of the config object is already set.

- ``peers.json``  : The current validator-set.

- ``genesis.peers.json`` : (optional, default peers.json) The initial
  validator-set of the network.

- ``cert.pem`` : (optional) The x509 certificate of the signaling server.

Please refer to the :ref:`usage` section for an explanation of the peers files.

When run as a standalone executable or from the mobile bindings, Babble will 
also look for a ``babble.toml`` file which is used to populate the Config 
object.