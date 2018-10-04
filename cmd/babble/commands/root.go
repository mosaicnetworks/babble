package commands

import (
	"github.com/spf13/cobra"
)

var (
	config = NewDefaultCLIConfig()
)

//RootCmd is the root command for Babble
var RootCmd = &cobra.Command{
	Use:              "babble",
	Short:            "babble consensus",
	TraverseChildren: true,
}
