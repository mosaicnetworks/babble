package mobile

import (
	"time"

	"github.com/mosaicnetworks/babble/src/config"
)

// MobileConfig ...
type MobileConfig struct {
	Heartbeat      int    //heartbeat timeout in milliseconds
	TCPTimeout     int    //TCP timeout in milliseconds
	MaxPool        int    //Max number of pooled connections
	CacheSize      int    //Number of items in LRU cache
	SyncLimit      int    //Max Events per sync
	EnableFastSync bool   //Enable fast sync
	Store          bool   //Use badger store
	LogLevel       string //debug, info, warn, error, fatal, panic
	Moniker        string //optional name
}

// NewMobileConfig ...
func NewMobileConfig(heartbeat int,
	tcpTimeout int,
	maxPool int,
	cacheSize int,
	syncLimit int,
	enableFastSync bool,
	store bool,
	logLevel string,
	moniker string) *MobileConfig {

	return &MobileConfig{
		Heartbeat:      heartbeat,
		TCPTimeout:     tcpTimeout,
		MaxPool:        maxPool,
		CacheSize:      cacheSize,
		SyncLimit:      syncLimit,
		EnableFastSync: enableFastSync,
		Store:          store,
		LogLevel:       logLevel,
		Moniker:        moniker,
	}
}

// DefaultMobileConfig ...
func DefaultMobileConfig() *MobileConfig {
	return &MobileConfig{
		Heartbeat:      10,
		TCPTimeout:     200,
		MaxPool:        2,
		CacheSize:      500,
		SyncLimit:      1000,
		EnableFastSync: true,
		Store:          false,
		LogLevel:       "debug",
	}
}

func (c *MobileConfig) toBabbleConfig() *config.Config {
	babbleConfig := config.NewDefaultConfig()

	babbleConfig.MaxPool = c.MaxPool
	babbleConfig.Store = c.Store
	babbleConfig.LogLevel = c.LogLevel
	babbleConfig.Moniker = c.Moniker
	babbleConfig.HeartbeatTimeout = time.Duration(c.Heartbeat) * time.Millisecond
	babbleConfig.TCPTimeout = time.Duration(c.TCPTimeout) * time.Millisecond
	babbleConfig.CacheSize = c.CacheSize
	babbleConfig.SyncLimit = c.SyncLimit
	babbleConfig.EnableFastSync = c.EnableFastSync
	babbleConfig.ServiceAddr = ""

	return babbleConfig
}
