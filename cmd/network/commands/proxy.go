package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/mosaicnetworks/babble/src/proxy/socket/babble"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var tx string

// ProxyCmd displays the version of babble being used
func NewProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Connect to a proxy",
		RunE:  connectProxy,
	}

	AddProxyFlags(cmd)

	return cmd
}

type Handler struct {
	stateHash []byte
}

// Called when a new block is comming
// You must provide a method to compute the stateHash incrementaly with incoming blocks
func (h *Handler) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	hash := h.stateHash

	for _, tx := range block.Transactions() {
		hash = crypto.SimpleHashFromTwoHashes(hash, crypto.SHA256(tx))

		fmt.Println(string(tx))
	}

	h.stateHash = hash

	response := proxy.CommitResponse{
		StateHash:            hash,
		InternalTransactions: block.InternalTransactions(),
	}

	return response, nil
}

// Called when syncing with the network
func (h *Handler) SnapshotHandler(blockIndex int) (snapshot []byte, err error) {
	return []byte{}, nil
}

// Called when syncing with the network
func (h *Handler) RestoreHandler(snapshot []byte) (stateHash []byte, err error) {
	return []byte{}, nil
}

func NewHandler() *Handler {
	return &Handler{}
}

func connectProxy(cmd *cobra.Command, args []string) error {
	i := config.Node

	fmt.Println("I !!!!!", i)

	babblePort := 1337 + (i * 10)
	proxyServPortStr := strconv.Itoa(babblePort + 1)
	proxyCliPortStr := strconv.Itoa(babblePort + 2)

	logger := logrus.New()

	logger.Level = logrus.InfoLevel

	proxy, err := babble.NewSocketBabbleProxy("127.0.0.1:"+proxyServPortStr, "127.0.0.1:"+proxyCliPortStr, NewHandler(), 1*time.Second, logger)
	if err != nil {
		panic(err)
	}

	if len(tx) > 0 {
		if err := proxy.SubmitTx([]byte(tx)); err != nil {
			panic(err)
		}

		return nil
	}

	if config.Stdin {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			if err := proxy.SubmitTx(scanner.Bytes()); err != nil {
				panic(err)
			}
		}

		return nil
	}

	for {
		time.Sleep(time.Second)
	}
	return nil
}

//AddRunFlags adds flags to the Run command
func AddProxyFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&config.Node, "node", config.Node, "Node index to connect to (starts from 0)")
	cmd.Flags().BoolVar(&config.Stdin, "stdin", config.Stdin, "Send some transactions from stdin")
	cmd.Flags().StringVar(&tx, "submit", tx, "Tx to submit and quit")
}
