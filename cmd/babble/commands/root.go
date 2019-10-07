package commands

import (
	"github.com/spf13/cobra"
)

var (
	_config = NewDefaultCLIConfig()
)

// RootCmd is the root command for Babble
var RootCmd = &cobra.Command{
	Use:              "babble",
	Short:            "babble consensus",
	TraverseChildren: true,
}
