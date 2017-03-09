/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"

	"github.com/arrivets/babble/proxy"
)

var (
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

	proxyAddress := c.String(ProxyAddressFlag.Name)
	clientAddress := c.String(ClientAddressFlag.Name)

	logger.WithFields(logrus.Fields{
		"proxy_addr":  proxyAddress,
		"client_addr": clientAddress,
	}).Debug("RUN")

	client, err := proxy.NewDummySocketClient(clientAddress, proxyAddress, logger)
	if err != nil {
		return err
	}

	// for {
	// 	ok, err := client.SubmitTx([]byte(fmt.Sprintf("Hello from %s\n", clientAddress)))
	// 	if err != nil {
	// 		if err != nil {
	// 			fmt.Printf("Error in SubmitTx: %v\n", err)
	// 		} else {
	// 			fmt.Printf("OK? %v\n", *ok)
	// 		}
	// 	}
	// 	time.Sleep(1 * time.Second)
	// }

	scanner := bufio.NewScanner(os.Stdin)
	var text string
	for text != "q" { // break the loop if text == "q"
		fmt.Print("Enter your text: ")
		scanner.Scan()
		text = scanner.Text()
		if text != "q" {
			fmt.Println("Your text was: ", text)
			ok, err := client.SubmitTx([]byte(text))
			if err != nil {
				fmt.Printf("Error in SubmitTx: %v\n", err)
			} else {
				fmt.Printf("OK? %v\n", *ok)
			}
		}
	}

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
