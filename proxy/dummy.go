package proxy

import (
	"fmt"
	"os"

	"time"

	"github.com/babbleio/babble/hashgraph"
	bproxy "github.com/babbleio/babble/proxy/babble"
	"github.com/sirupsen/logrus"
)

type State struct {
	logger *logrus.Logger
}

func (a *State) CommitBlock(block hashgraph.Block) error {
	a.logger.WithField("block", block).Debug("CommitBlock")
	return a.writeBlock(block)
}

func (a *State) writeBlock(block hashgraph.Block) error {
	file, err := a.getFile()
	if err != nil {
		a.logger.Error(err)
		return err
	}
	defer file.Close()

	// write some text to file
	for _, tx := range block.Transactions() {
		_, err = file.WriteString(fmt.Sprintf("%s\n", string(tx)))
		if err != nil {
			a.logger.Error(err)
			return err
		}
	}

	err = file.Sync()
	if err != nil {
		a.logger.Error(err)
		return err
	}

	return nil
}

func (a *State) writeMessage(tx []byte) {
	file, err := a.getFile()
	if err != nil {
		a.logger.Error(err)
		return
	}
	defer file.Close()

	// write some text to file
	_, err = file.WriteString(fmt.Sprintf("%s\n", string(tx)))
	if err != nil {
		a.logger.Error(err)
	}
	err = file.Sync()
	if err != nil {
		a.logger.Error(err)
	}
}

func (a *State) getFile() (*os.File, error) {
	path := "messages.txt"
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
}

//------------------------------------------------------

type DummySocketClient struct {
	state       *State
	babbleProxy *bproxy.SocketBabbleProxy
	logger      *logrus.Logger
}

func NewDummySocketClient(clientAddr string, nodeAddr string, logger *logrus.Logger) (*DummySocketClient, error) {

	babbleProxy, err := bproxy.NewSocketBabbleProxy(nodeAddr, clientAddr, 1*time.Second)
	if err != nil {
		return nil, err
	}

	state := State{logger: logger}
	state.writeMessage([]byte(clientAddr))

	client := &DummySocketClient{
		state:       &state,
		babbleProxy: babbleProxy,
		logger:      logger,
	}

	go client.Run()

	return client, nil
}

func (c *DummySocketClient) Run() {
	for {
		select {
		case block := <-c.babbleProxy.CommitCh():
			c.logger.Debug("CommitBlock")
			c.state.CommitBlock(block)
		}
	}
}

func (c *DummySocketClient) SubmitTx(tx []byte) error {
	return c.babbleProxy.SubmitTx(tx)
}
