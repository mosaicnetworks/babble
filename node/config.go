package node

import (
	"testing"
	"time"

	"github.com/babbleio/babble/common"
	"github.com/Sirupsen/logrus"
)

type Config struct {
	HeartbeatTimeout time.Duration
	TCPTimeout       time.Duration
	CacheSize        int
	SyncLimit        int
	Logger           *logrus.Logger
}

func NewConfig(heartbeat time.Duration,
	timeout time.Duration,
	cacheSize int,
	syncLimit int,
	logger *logrus.Logger) *Config {
	return &Config{
		HeartbeatTimeout: heartbeat,
		TCPTimeout:       timeout,
		CacheSize:        cacheSize,
		SyncLimit:        syncLimit,
		Logger:           logger,
	}
}

func DefaultConfig() *Config {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	return &Config{
		HeartbeatTimeout: 1000 * time.Millisecond,
		TCPTimeout:       1000 * time.Millisecond,
		CacheSize:        500,
		SyncLimit:        100,
		Logger:           logger,
	}
}

func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.Logger = common.NewTestLogger(t)
	return config
}
