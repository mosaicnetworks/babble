package babble

import (
	"crypto/ecdsa"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

type BabbleConfig struct {
	NodeConfig node.Config `mapstructure:",squash"`

	DataDir     string `mapstructure:"datadir"`
	BindAddr    string `mapstructure:"listen"`
	ServiceAddr string `mapstructure:"service-listen"`
	MaxPool     int    `mapstructure:"max-pool"`
	Store       bool   `mapstructure:"store"`
	LogLevel    string `mapstructure:"log"`

	LoadPeers bool
	Proxy     proxy.AppProxy
	Key       *ecdsa.PrivateKey
	Logger    *logrus.Logger
}

func NewDefaultConfig() *BabbleConfig {
	config := &BabbleConfig{
		DataDir:    DefaultDataDir(),
		BindAddr:   ":1337",
		Proxy:      nil,
		Logger:     logrus.New(),
		MaxPool:    2,
		NodeConfig: *node.DefaultConfig(),
		Store:      false,
		LoadPeers:  true,
		Key:        nil,
	}

	config.NodeConfig.Logger = config.Logger

	return config
}

func (c *BabbleConfig) BadgerDir() string {
	return filepath.Join(c.DataDir, "badger_db")
}

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

func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

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
