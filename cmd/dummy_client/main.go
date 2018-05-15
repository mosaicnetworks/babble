package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"

	"github.com/mosaicnetworks/babble/proxy"
)

var (
	NameFlag = cli.StringFlag{
		Name:  "name",
		Usage: "Client Name",
	}
	ProxyAddressFlag = cli.StringFlag{
		Name:  "proxy_addr",
		Usage: "IP:Port to bind Proxy Server",
		Value: "127.0.0.1:1338",
	}
	ClientAddressFlag = cli.StringFlag{
		Name:  "client_addr",
		Usage: "IP:Port of Client App",
		Value: "127.0.0.1:1339",
	}
	LogLevelFlag = cli.StringFlag{
		Name:  "log_level",
		Usage: "debug, info, warn, error, fatal, panic",
		Value: "debug",
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "dummy"
	app.Usage = "Dummy Socket Client for Babble"
	app.Flags = []cli.Flag{
		NameFlag,
		ProxyAddressFlag,
		ClientAddressFlag,
		LogLevelFlag,
	}
	app.Action = run
	app.Run(os.Args)
}

func run(c *cli.Context) error {
	logger := logrus.New()
	logger.Level = logLevel(c.String(LogLevelFlag.Name))

	name := c.String(NameFlag.Name)
	proxyAddress := c.String(ProxyAddressFlag.Name)
	clientAddress := c.String(ClientAddressFlag.Name)

	logger.WithFields(logrus.Fields{
		"name":        name,
		"proxy_addr":  proxyAddress,
		"client_addr": clientAddress,
	}).Debug("RUN")

	client, err := proxy.NewDummySocketClient(clientAddress, proxyAddress, logger)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Print("Enter your text: ")
		text := scanner.Text()
		message := fmt.Sprintf("%s: %s", name, text)
		if err := client.SubmitTx([]byte(message)); err != nil {
			fmt.Printf("Error in SubmitTx: %v\n", err)
		}
	}

	select {}

	return nil
}

func logLevel(l string) logrus.Level {
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
