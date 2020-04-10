package main

import (
	"os"

	cmd "github.com/mosaicnetworks/babble/cmd/signal/commands"
)

func main() {
	rootCmd := cmd.RootCmd

	//Do not print usage when error occurs
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
