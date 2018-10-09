package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mosaicnetworks/babble/src/dummy"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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

	logger := newLogger()
	logger.Level = logLevel(c.String(LogLevelFlag.Name))

	name := c.String(NameFlag.Name)
	proxyAddress := c.String(ProxyAddressFlag.Name)
	clientAddress := c.String(ClientAddressFlag.Name)

	logger.WithFields(logrus.Fields{
		"name":        name,
		"proxy_addr":  proxyAddress,
		"client_addr": clientAddress,
	}).Debug("RUN")

	//Create and run Dummy Socket Client
	client, err := dummy.NewDummySocketClient(clientAddress, proxyAddress, logger)
	if err != nil {
		return err
	}

	//Listen for input messages from tty
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Print("Enter your text: ")
		text := scanner.Text()
		message := fmt.Sprintf("%s: %s", name, text)
		if err := client.SubmitTx([]byte(message)); err != nil {
			fmt.Printf("Error in SubmitTx: %v\n", err)
		}
	}

	return nil
}

func newLogger() *logrus.Logger {
	logger := logrus.New()

	pathMap := lfshook.PathMap{}

	_, err := os.OpenFile("info.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logger.Info("Failed to open info.log file, using default stderr")
	} else {
		pathMap[logrus.InfoLevel] = "info.log"
	}

	_, err = os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logger.Info("Failed to open debug.log file, using default stderr")
	} else {
		pathMap[logrus.DebugLevel] = "debug.log"
	}

	if err == nil {
		logger.Out = ioutil.Discard
	}

	logger.Hooks.Add(lfshook.NewHook(
		pathMap,
		&logrus.TextFormatter{},
	))

	return logger
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
