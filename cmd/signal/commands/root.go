package commands

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
	"github.com/spf13/cobra"
)

var url = "localhost:8000"
var realm = "office"

//RootCmd is the root command for the signaling server
var RootCmd = &cobra.Command{
	Use:   "signal",
	Short: "WebRTC signaling server using WebSockets",
	RunE:  runServer,
}

// runServer starts the WAMP server and waits for a SIGINT or SIGTERM
func runServer(cmd *cobra.Command, args []string) error {
	server, err := wamp.NewServer(url, realm)
	if err != nil {
		log.Fatal(err)
	}

	go server.Run()

	//Prepare sigCh to relay SIGINT and SIGTERM system calls
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	server.Shutdown()

	return nil
}
