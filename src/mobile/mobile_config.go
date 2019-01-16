package mobile

import (
	"time"

	"github.com/mosaicnetworks/babble/src/babble"
)

type MobileConfig struct {
	Heartbeat  int    //heartbeat timeout in milliseconds
	TCPTimeout int    //TCP timeout in milliseconds
	MaxPool    int    //Max number of pooled connections
	CacheSize  int    //Number of items in LRU cache
	SyncLimit  int    //Max Events per sync
	Store      bool   //Use badger store
	LogLevel   string //debug, info, warn, error, fatal, panic

}

func NewMobileConfig(heartbeat int,
	tcpTimeout int,
	maxPool int,
	cacheSize int,
	syncLimit int,
	store bool,
	logLevel string) *MobileConfig {

	return &MobileConfig{
		Heartbeat:  heartbeat,
		TCPTimeout: tcpTimeout,
		MaxPool:    maxPool,
		CacheSize:  cacheSize,
		SyncLimit:  syncLimit,
		Store:      store,
		LogLevel:   logLevel,
	}
}

func DefaultMobileConfig() *MobileConfig {
	return &MobileConfig{
		Heartbeat:  1000,
		TCPTimeout: 1000,
		MaxPool:    2,
		CacheSize:  500,
		SyncLimit:  1000,
		Store:      false,
		LogLevel:   "debug",
	}
}

func (c *MobileConfig) toBabbleConfig() *babble.BabbleConfig {
	babbleConfig := babble.NewDefaultConfig()

	babbleConfig.Logger.SetLevel(babble.LogLevel(c.LogLevel))

	babbleConfig.MaxPool = c.MaxPool
	babbleConfig.Store = c.Store
	babbleConfig.LogLevel = c.LogLevel

	babbleConfig.NodeConfig.HeartbeatTimeout = time.Duration(c.Heartbeat) * time.Millisecond
	babbleConfig.NodeConfig.TCPTimeout = time.Duration(c.TCPTimeout) * time.Millisecond
	babbleConfig.NodeConfig.CacheSize = c.CacheSize
	babbleConfig.NodeConfig.SyncLimit = c.SyncLimit

	return babbleConfig
}
