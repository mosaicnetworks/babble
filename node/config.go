package node

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/babbleio/babble/common"
)

type Config struct {
	HeartbeatTimeout time.Duration
	TCPTimeout       time.Duration
	CacheSize        int
	SyncLimit        int
	DBPath           string
	Logger           *logrus.Logger
}

func NewConfig(heartbeat time.Duration,
	timeout time.Duration,
	cacheSize int,
	syncLimit int,
	dbPath string,
	logger *logrus.Logger) *Config {
	return &Config{
		HeartbeatTimeout: heartbeat,
		TCPTimeout:       timeout,
		CacheSize:        cacheSize,
		SyncLimit:        syncLimit,
		DBPath:           dbPath,
		Logger:           logger,
	}
}

func DefaultConfig() *Config {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	dbPath, _ := ioutil.TempDir("", "badger")
	return &Config{
		HeartbeatTimeout: 1000 * time.Millisecond,
		TCPTimeout:       1000 * time.Millisecond,
		CacheSize:        500,
		SyncLimit:        100,
		DBPath:           dbPath,
		Logger:           logger,
	}
}

func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.Logger = common.NewTestLogger(t)
	return config
}
