package node

import (
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/sirupsen/logrus"
)

//Config is a Configuration Object Definition
type Config struct {
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat"`
	TCPTimeout       time.Duration `mapstructure:"timeout"`
	JoinTimeout      time.Duration `mapstructure:"join_timeout"`
	CacheSize        int           `mapstructure:"cache-size"`
	SyncLimit        int           `mapstructure:"sync-limit"`
	EnableFastSync   bool          `mapstructure:"enable-fast-sync"`
	Bootstrap        bool          `mapstructure:"bootstrap"`
	Logger           *logrus.Logger
}

//NewConfig eturns a new Config Object
func NewConfig(heartbeat time.Duration,
	timeout time.Duration,
	joinTimeout time.Duration,
	cacheSize int,
	syncLimit int,
	enableFastSync bool,
	logger *logrus.Logger) *Config {

	return &Config{
		HeartbeatTimeout: heartbeat,
		TCPTimeout:       timeout,
		JoinTimeout:      joinTimeout,
		CacheSize:        cacheSize,
		SyncLimit:        syncLimit,
		EnableFastSync:   enableFastSync,
		Logger:           logger,
	}
}

//DefaultConfig returns a Default Config Object
func DefaultConfig() *Config {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	return &Config{
		HeartbeatTimeout: 10 * time.Millisecond,
		TCPTimeout:       1000 * time.Millisecond,
		JoinTimeout:      10000 * time.Millisecond,
		CacheSize:        5000,
		SyncLimit:        1000,
		EnableFastSync:   true,
		Logger:           logger,
	}
}

//TestConfig returns a Preset Test Configuration
func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.Logger = common.NewTestLogger(t)
	return config
}
