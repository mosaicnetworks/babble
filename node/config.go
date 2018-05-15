package node

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/common"
	"github.com/sirupsen/logrus"
)

type Config struct {
	HeartbeatTimeout time.Duration
	TCPTimeout       time.Duration
	CacheSize        int
	SyncLimit        int
	StoreType        string
	StorePath        string
	Logger           *logrus.Logger
}

func NewConfig(heartbeat time.Duration,
	timeout time.Duration,
	cacheSize int,
	syncLimit int,
	storeType string,
	storePath string,
	logger *logrus.Logger) *Config {
	return &Config{
		HeartbeatTimeout: heartbeat,
		TCPTimeout:       timeout,
		CacheSize:        cacheSize,
		SyncLimit:        syncLimit,
		StoreType:        storeType,
		StorePath:        storePath,
		Logger:           logger,
	}
}

func DefaultConfig() *Config {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	storeType := "badger"
	storePath, _ := ioutil.TempDir("", "badger")
	return &Config{
		HeartbeatTimeout: 1000 * time.Millisecond,
		TCPTimeout:       1000 * time.Millisecond,
		CacheSize:        500,
		SyncLimit:        100,
		StoreType:        storeType,
		StorePath:        storePath,
		Logger:           logger,
	}
}

func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.StoreType = "inmem"
	config.Logger = common.NewTestLogger(t)
	return config
}
