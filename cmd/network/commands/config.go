package commands

import "github.com/mosaicnetworks/babble/src/babble"

//CLIConfig contains configuration for the Run command
type CLIConfig struct {
	Babble  babble.BabbleConfig `mapstructure:",squash"`
	NbNodes int                 `mapstructure:"nodes"`
	SendTxs int                 `mapstructure:"send-txs"`
	Stdin   bool                `mapstructure:"stdin"`
	Node    int                 `mapstructure:"node"`
}

//NewDefaultCLIConfig creates a CLIConfig with default values
func NewDefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Babble:  *babble.NewDefaultConfig(),
		NbNodes: 4,
		SendTxs: 0,
		Stdin:   false,
		Node:    0,
	}
}
