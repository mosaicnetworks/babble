package commands

import (
	"github.com/mosaicnetworks/babble/src/config"
)

// CLIConfig contains configuration for the Run command
type CLIConfig struct {
	Babble     config.Config `mapstructure:",squash"`
	ProxyAddr  string        `mapstructure:"proxy-listen"`
	ClientAddr string        `mapstructure:"client-connect"`
}

// NewDefaultCLIConfig creates a CLIConfig with default values
func NewDefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Babble:     *config.NewDefaultConfig(),
		ProxyAddr:  "127.0.0.1:1338",
		ClientAddr: "127.0.0.1:1339",
	}
}
