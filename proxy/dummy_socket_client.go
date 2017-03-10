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
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"time"

	"github.com/Sirupsen/logrus"
)

type State struct {
	logger *logrus.Logger
}

func (a *State) CommitTx(tx []byte, ack *bool) error {
	a.logger.WithField("Tx", string(tx)).Debug("CommitTx")
	a.writeMessage(tx)
	*ack = true
	return nil
}

func (a *State) writeMessage(tx []byte) {
	file, err := a.getFile()
	if err != nil {
		a.logger.Error(err)
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
	nodeAddr    string
	netListener *net.Listener
	rpcServer   *rpc.Server
	consumeCh   chan []byte
	logger      *logrus.Logger
}

func NewDummySocketClient(clientAddr string, nodeAddr string, logger *logrus.Logger) (*DummySocketClient, error) {
	rpcServer := rpc.NewServer()
	state := &State{logger: logger}
	state.writeMessage([]byte(clientAddr))
	rpcServer.Register(&State{logger: logger})
	l, err := net.Listen("tcp", clientAddr)
	if err != nil {
		return nil, err
	}
	client := &DummySocketClient{
		nodeAddr:    nodeAddr,
		netListener: &l,
		rpcServer:   rpcServer,
		consumeCh:   make(chan []byte),
		logger:      logger,
	}

	go client.listen()
	return client, nil
}

func (c *DummySocketClient) listen() {
	for {
		conn, err := (*c.netListener).Accept()
		if err != nil {
			c.logger.WithField("error", err).Error("Failed to accept")
		}

		go (*c.rpcServer).ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func (c *DummySocketClient) getConnection() (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", c.nodeAddr, 1*time.Second)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

func (c *DummySocketClient) SubmitTx(tx []byte) (*bool, error) {
	rpcConn, err := c.getConnection()
	if err != nil {
		return nil, err
	}
	var ack bool
	err = rpcConn.Call("Babble.SubmitTx", tx, &ack)
	if err != nil {
		return nil, err
	}
	return &ack, nil
}

func (c *DummySocketClient) Consumer() chan []byte {
	return c.consumeCh
}
