package commands

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/mosaicnetworks/babble/src/babble"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//NewRunCmd returns the command that starts a Babble node
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run node",
		PreRunE: loadConfig,
		RunE:    runBabbles,
	}

	AddRunFlags(cmd)

	return cmd
}

/*******************************************************************************
* RUN
*******************************************************************************/

func buildConfig() error {
	babblePort := 1337

	peersJSON := `[`

	for i := 0; i < config.NbNodes; i++ {
		nb := strconv.Itoa(i)

		babblePortStr := strconv.Itoa(babblePort + (i * 10))

		babbleNode := exec.Command("babble", "keygen", "--pem=/tmp/babble_configs/.babble"+nb+"/priv_key.pem", "--pub=/tmp/babble_configs/.babble"+nb+"/key.pub")

		res, err := babbleNode.CombinedOutput()
		if err != nil {
			log.Fatal(err, res)
		}

		pubKey, err := ioutil.ReadFile("/tmp/babble_configs/.babble" + nb + "/key.pub")
		if err != nil {
			log.Fatal(err, res)
		}

		peersJSON += `	{
		"NetAddr":"127.0.0.1:` + babblePortStr + `",
		"PubKeyHex":"` + string(pubKey) + `"
	},
`
	}

	peersJSON = peersJSON[:len(peersJSON)-2]
	peersJSON += `
]
`

	for i := 0; i < config.NbNodes; i++ {
		nb := strconv.Itoa(i)

		err := ioutil.WriteFile("/tmp/babble_configs/.babble"+nb+"/peers.json", []byte(peersJSON), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func sendTxs(babbleNode *exec.Cmd, i int) {
	ticker := time.NewTicker(1 * time.Second)
	nb := strconv.Itoa(i)

	txNb := 0

	for range ticker.C {
		if txNb == config.SendTxs {
			ticker.Stop()

			break
		}

		network := exec.Command("network", "proxy", "--node="+nb, "--submit="+nb+"_"+strconv.Itoa(txNb))

		err := network.Run()
		if err != nil {
			continue
		}

		txNb++
	}
}

func runBabbles(cmd *cobra.Command, args []string) error {
	os.RemoveAll("/tmp/babble_configs")

	if err := buildConfig(); err != nil {
		log.Fatal(err)
	}

	babblePort := 1337
	servicePort := 8080

	wg := sync.WaitGroup{}

	var processes = make([]*os.Process, config.NbNodes)

	for i := 0; i < config.NbNodes; i++ {
		wg.Add(1)

		go func(i int) {
			nb := strconv.Itoa(i)
			babblePortStr := strconv.Itoa(babblePort + (i * 10))
			proxyServPortStr := strconv.Itoa(babblePort + (i * 10) + 1)
			proxyCliPortStr := strconv.Itoa(babblePort + (i * 10) + 2)

			servicePort := strconv.Itoa(servicePort + i)

			defer wg.Done()

			babbleNode := exec.Command("babble", "run", "-l=127.0.0.1:"+babblePortStr, "--datadir=/tmp/babble_configs/.babble"+nb, "--proxy-listen=127.0.0.1:"+proxyServPortStr, "--client-connect=127.0.0.1:"+proxyCliPortStr, "-s=127.0.0.1:"+servicePort, "--heartbeat="+config.Babble.NodeConfig.HeartbeatTimeout.String())
			err := babbleNode.Start()

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Running", i)

			if config.SendTxs > 0 {
				go sendTxs(babbleNode, i)
			}

			processes[i] = babbleNode.Process

			babbleNode.Wait()

			fmt.Println("Terminated", i)

		}(i)
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for range c {
			for _, proc := range processes {
				proc.Kill()
			}
		}
	}()

	wg.Wait()

	return nil
}

/*******************************************************************************
* CONFIG
*******************************************************************************/

//AddRunFlags adds flags to the Run command
func AddRunFlags(cmd *cobra.Command) {
	cmd.Flags().Int("nodes", config.NbNodes, "Amount of nodes to spawn")
	cmd.Flags().String("datadir", config.Babble.DataDir, "Top-level directory for configuration and data")
	cmd.Flags().String("log", config.Babble.LogLevel, "debug, info, warn, error, fatal, panic")
	cmd.Flags().Duration("heartbeat", config.Babble.NodeConfig.HeartbeatTimeout, "Time between gossips")

	cmd.Flags().Int("sync-limit", config.Babble.NodeConfig.SyncLimit, "Max number of events for sync")
	cmd.Flags().Int("send-txs", config.SendTxs, "Send some random transactions")
}

func loadConfig(cmd *cobra.Command, args []string) error {

	err := bindFlagsLoadViper(cmd)
	if err != nil {
		return err
	}

	config, err = parseConfig()
	if err != nil {
		return err
	}

	config.Babble.Logger.Level = babble.LogLevel(config.Babble.LogLevel)
	config.Babble.NodeConfig.Logger = config.Babble.Logger

	return nil
}

//Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command) error {
	// cmd.Flags() includes flags from this command and all persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	viper.SetConfigName("babble")              // name of config file (without extension)
	viper.AddConfigPath(config.Babble.DataDir) // search root directory
	// viper.AddConfigPath(filepath.Join(config.Babble.DataDir, "babble")) // search root directory /config

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		config.Babble.Logger.Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		config.Babble.Logger.Debugf("No config file found in: %s", config.Babble.DataDir)
	} else {
		return err
	}

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
