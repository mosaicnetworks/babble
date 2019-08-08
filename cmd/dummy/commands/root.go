package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mosaicnetworks/babble/src/proxy/dummy"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config = NewDefaultCLIConfig()
	logger *logrus.Logger
)

func init() {
	RootCmd.Flags().String("name", config.Name, "Client name")
	RootCmd.Flags().String("client-listen", config.ClientAddr, "Listen IP:Port of Dummy Socket Client")
	RootCmd.Flags().String("proxy-connect", config.ProxyAddr, "IP:Port to connect to Babble proxy")
	RootCmd.Flags().Bool("discard", config.Discard, "discard output to stderr and sdout")
	RootCmd.Flags().String("log", config.LogLevel, "debug, info, warn, error, fatal, panic")
}

//RootCmd is the root command for Dummy
var RootCmd = &cobra.Command{
	Use:     "dummy",
	Short:   "Dummy Socket Client for Babble",
	PreRunE: loadConfig,
	RunE:    runDummy,
}

/*******************************************************************************
* RUN
*******************************************************************************/

func runDummy(cmd *cobra.Command, args []string) error {

	name := config.Name
	proxyAddress := config.ProxyAddr
	clientAddress := config.ClientAddr

	//Create and run Dummy Socket Client
	client, err := dummy.NewDummySocketClient(clientAddress,
		proxyAddress,
		logger.WithField("component", "DUMMY"))
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

/*******************************************************************************
* CONFIG
*******************************************************************************/

func loadConfig(cmd *cobra.Command, args []string) error {

	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return err
	}

	config, err = parseConfig()
	if err != nil {
		return err
	}

	logger = newLogger()
	logger.Level = logLevel(config.LogLevel)

	logger.WithFields(logrus.Fields{
		"name":          config.Name,
		"client-listen": config.ClientAddr,
		"proxy-connect": config.ProxyAddr,
		"discard":       config.Discard,
		"log":           config.LogLevel,
	}).Debug("RUN")

	return nil
}

//Retrieve the default environment configuration.
func parseConfig() (*CLIConfig, error) {
	conf := NewDefaultCLIConfig()
	err := viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}

func newLogger() *logrus.Logger {
	logger := logrus.New()

	pathMap := lfshook.PathMap{}

	_, err := os.OpenFile("dummy_info.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logger.Info("Failed to open dummy_info.log file, using default stderr")
	} else {
		pathMap[logrus.InfoLevel] = "dummy_info.log"
	}

	_, err = os.OpenFile("dummy_debug.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logger.Info("Failed to open dummy_debug.log file, using default stderr")
	} else {
		pathMap[logrus.DebugLevel] = "dummy_debug.log"
	}

	if err == nil && config.Discard {
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
