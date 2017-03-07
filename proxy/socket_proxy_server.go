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

	"github.com/Sirupsen/logrus"
)

type SocketProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	consumeCh   chan []byte
	logger      *logrus.Logger
}

func NewSocketProxyServer(bindAddress string, logger *logrus.Logger) *SocketProxyServer {
	server := &SocketProxyServer{
		consumeCh: make(chan []byte),
		logger:    logger,
	}
	server.register(bindAddress)
	return server
}

func (p *SocketProxyServer) register(bindAddress string) {
	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("Babble", p)
	p.rpcServer = rpcServer

	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		p.logger.WithField("error", err).Error("Failed to listen")
	}
	p.netListener = &l
}

func (p *SocketProxyServer) listen() {
	for {
		conn, err := (*p.netListener).Accept()
		if err != nil {
			p.logger.WithField("error", err).Error("Failed to accept")
		}

		go (*p.rpcServer).ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func (p *SocketProxyServer) SubmitTx(tx []byte, ack *bool) error {
	p.logger.Debug("SubmitTx")
	p.consumeCh <- tx
	*ack = true
	return nil
}
