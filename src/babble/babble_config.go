package babble

import (
	"crypto/ecdsa"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/proxy"
	sproxy "github.com/mosaicnetworks/babble/src/proxy/socket/app"
	"github.com/sirupsen/logrus"
)

type BabbleConfig struct {
	DataDir     string
	BindAddr    string
	ServiceAddr string
	Logger      *logrus.Logger
	MaxPool     int
	NodeConfig  *node.Config
	Proxy       proxy.AppProxy
	Store       bool
	LoadPeers   bool
	Key         *ecdsa.PrivateKey
}

func NewDefaultConfig() *BabbleConfig {
	config := &BabbleConfig{
		DataDir:    DefaultDataDir(),
		BindAddr:   ":1337",
		Proxy:      nil,
		Logger:     logrus.New(),
		MaxPool:    2,
		NodeConfig: node.DefaultConfig(),
		Store:      false,
		LoadPeers:  true,
		Key:        nil,
	}

	config.NodeConfig.Logger = config.Logger

	//XXX
	config.Proxy, _ = sproxy.NewSocketAppProxy("127.0.0.1:1338", "127.0.0.1:1339", 1*time.Second, config.Logger)

	return config
}

func DefaultBadgerDir() string {
	dataDir := DefaultDataDir()
	if dataDir != "" {
		return filepath.Join(dataDir, "badger_db")
	}
	return ""
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
