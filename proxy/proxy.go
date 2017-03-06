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
	"time"

	"fmt"

	"github.com/Sirupsen/logrus"
)

type Proxy struct {
	clientAddress string
	bindAddress   string

	client *ProxyClient
	server *ProxyServer

	logger *logrus.Logger
}

func NewProxy(clientAddr string, bindAddr string, timeout time.Duration, logger *logrus.Logger) *Proxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	client := NewProxyClient(clientAddr, timeout, logger)
	server := NewProxyServer(bindAddr, logger)

	proxy := &Proxy{
		clientAddress: clientAddr,
		bindAddress:   bindAddr,
		client:        client,
		server:        server,
		logger:        logger,
	}
	go proxy.server.listen()

	return proxy
}

func (p *Proxy) Consumer() chan []byte {
	return p.server.consumeCh
}

func (p *Proxy) CommitTx(tx []byte) error {
	ack, err := p.client.CommitTx(tx)
	if err != nil {
		return err
	}
	if !*ack {
		return fmt.Errorf("App returned false to CommitTx")
	}
	return nil
}
