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
package proxy

import (
	"fmt"
	"os"

	"time"

	"github.com/Sirupsen/logrus"
	bproxy "github.com/babbleio/babble/proxy/babble"
)

type State struct {
	logger *logrus.Logger
}

func (a *State) CommitTx(tx []byte) error {
	a.logger.WithField("Tx", string(tx)).Debug("CommitTx")
	a.writeMessage(tx)
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
		case tx := <-c.babbleProxy.CommitCh():
			c.logger.Debug("CommitTx")
			c.state.CommitTx(tx)
		}
	}
}

func (c *DummySocketClient) SubmitTx(tx []byte) error {
	return c.babbleProxy.SubmitTx(tx)
}
