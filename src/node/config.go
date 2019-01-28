package node

import (
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/sirupsen/logrus"
)

type Config struct {
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat"`
	TCPTimeout       time.Duration `mapstructure:"timeout"`
	CacheSize        int           `mapstructure:"cache-size"`
	SyncLimit        int           `mapstructure:"sync-limit"`
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
		HeartbeatTimeout: 10 * time.Millisecond,
		TCPTimeout:       1000 * time.Millisecond,
		CacheSize:        5000,
		SyncLimit:        1000,
		Logger:           logger,
	}
}

func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.Logger = common.NewTestLogger(t)
	return config
}
