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
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"time"

	"github.com/Sirupsen/logrus"
)

type State struct {
	logger *logrus.Logger
}

func (a *State) CommitTx(tx []byte, ack *bool) error {
	a.logger.WithField("Tx", string(tx)).Debug("CommitTx")
	*ack = true
	return nil
}

//------------------------------------------------------

type DummyClient struct {
	nodeAddr    string
	netListener *net.Listener
	rpcServer   *rpc.Server
	consumeCh   chan []byte
	logger      *logrus.Logger
}

func NewDummyClient(clientAddr string, nodeAddr string, logger *logrus.Logger) (*DummyClient, error) {
	rpcServer := rpc.NewServer()
	rpcServer.Register(&State{logger: logger})
	l, err := net.Listen("tcp", clientAddr)
	if err != nil {
		return nil, err
	}
	client := &DummyClient{
		nodeAddr:    nodeAddr,
		netListener: &l,
		rpcServer:   rpcServer,
		consumeCh:   make(chan []byte),
		logger:      logger,
	}

	go client.listen()
	return client, nil
}

func (c *DummyClient) listen() {
	for {
		conn, err := (*c.netListener).Accept()
		if err != nil {
			c.logger.WithField("error", err).Error("Failed to accept")
		}

		go (*c.rpcServer).ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func (c *DummyClient) getConnection() (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", c.nodeAddr, 1*time.Second)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

func (c *DummyClient) SubmitTx(tx []byte) (*bool, error) {
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

func (c *DummyClient) Consumer() chan []byte {
	return c.consumeCh
}
