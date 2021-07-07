// Package babble implements a component that the TCP AppProxy can connect to.
package babble

import (
	"fmt"
	"time"

	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

// SocketBabbleProxy is a Golang implementation of a service that binds to a
// remote Babble over an RPC/TCP connection. It implements handlers for the RPC
// requests sent by the SocketAppProxy, and submits transactions to Babble via
// an RPC request. A SocketBabbleProxy can be implemented in any programming
// language as long as it implements the AppProxy interface over RPC.
type SocketBabbleProxy struct {
	nodeAddress string
	bindAddress string

	handler proxy.ProxyHandler

	client *SocketBabbleProxyClient
	server *SocketBabbleProxyServer
}

// NewSocketBabbleProxy creates a new SocketBabbleProxy
func NewSocketBabbleProxy(
	nodeAddr string,
	bindAddr string,
	handler proxy.ProxyHandler,
	timeout time.Duration,
	logger *logrus.Entry,
) (*SocketBabbleProxy, error) {

	if logger == nil {
		log := logrus.New()
		log.Level = logrus.DebugLevel
		logger = logrus.NewEntry(log)
	}

	client := NewSocketBabbleProxyClient(nodeAddr, timeout)

	server, err := NewSocketBabbleProxyServer(bindAddr, handler, timeout, logger)

	if err != nil {
		return nil, err
	}

	proxy := &SocketBabbleProxy{
		nodeAddress: nodeAddr,
		bindAddress: bindAddr,
		handler:     handler,
		client:      client,
		server:      server,
	}

	go proxy.server.listen()

	return proxy, nil
}

// SubmitTx submits a transaction to Babble
func (p *SocketBabbleProxy) SubmitTx(tx []byte) error {
	ack, err := p.client.SubmitTx(tx)
	if err != nil {
		return err
	}

	if !*ack {
		return fmt.Errorf("Failed to deliver transaction to Babble")
	}

	return nil
}
