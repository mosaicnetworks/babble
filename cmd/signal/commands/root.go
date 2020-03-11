package commands

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var url = ":443"
var realm = "office"
var certFile = "cert.pem"
var keyFile = "key.pem"

func init() {
	RootCmd.Flags().StringVar(&url, "url", url, "Listen IP:Port")
	RootCmd.Flags().StringVar(&realm, "realm", realm, "Administrative routing domain within the WebRTC signaling")
	RootCmd.Flags().StringVar(&certFile, "cert-file", certFile, "File containing TLS certificate")
	RootCmd.Flags().StringVar(&certFile, "key-file", certFile, "File containing certificate key")
	viper.BindPFlags(RootCmd.Flags())
}

//RootCmd is the root command for the signaling server
var RootCmd = &cobra.Command{
	Use:   "signal",
	Short: "WebRTC signaling server using WebSocket Secure",
	RunE:  runServer,
}

// runServer starts the WAMP server and waits for a SIGINT or SIGTERM
func runServer(cmd *cobra.Command, args []string) error {
	server, err := wamp.NewServer(url,
		realm,
		certFile,
		keyFile,
		logrus.New().WithField("component", "signal-server"))
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
