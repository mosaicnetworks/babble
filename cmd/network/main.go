package main

import (
	_ "net/http/pprof"
	"os"

	cmd "github.com/mosaicnetworks/babble/cmd/network/commands"
)

func main() {
	rootCmd := cmd.RootCmd

	rootCmd.AddCommand(
		cmd.VersionCmd,
		cmd.NewProxyCmd(),
		cmd.NewLog(),
		cmd.NewRunCmd())

	//Do not print usage when error occurs
	rootCmd.SilenceUsage = true

	// signalChan := make(chan os.Signal, 1)
	// cleanupDone := make(chan struct{})

	// signal.Notify(signalChan, os.Interrupt)

	// go func() {
	// 	<-signalChan

	// 	fmt.Println("\nReceived an interrupt, stopping services...\n")

	// 	// cleanup(services, c)
	// }()

	// go func() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	// }()

	// <-cleanupDone
}
