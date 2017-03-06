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

type ProxyClient struct {
	nodeAddr string
	timeout  time.Duration
	logger   *logrus.Logger
}

func NewProxyClient(nodeAddr string, timeout time.Duration, logger *logrus.Logger) *ProxyClient {
	return &ProxyClient{
		nodeAddr: nodeAddr,
		timeout:  timeout,
		logger:   logger,
	}
}

func (p *ProxyClient) getConnection() (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", p.nodeAddr, p.timeout)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

func (p *ProxyClient) CommitTx(tx []byte) (*bool, error) {
	rpcConn, err := p.getConnection()
	if err != nil {
		return nil, err
	}
	var ack bool
	err = rpcConn.Call("State.CommitTx", tx, &ack)
	if err != nil {
		return nil, err
	}
	return &ack, nil
}
