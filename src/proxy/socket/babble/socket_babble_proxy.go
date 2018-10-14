package babble

import (
	"fmt"
	"time"

	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

type SocketBabbleProxy struct {
	nodeAddress string
	bindAddress string

	handler proxy.ProxyHandler

	client *SocketBabbleProxyClient
	server *SocketBabbleProxyServer
}

func NewSocketBabbleProxy(
	nodeAddr string,
	bindAddr string,
	handler proxy.ProxyHandler,
	timeout time.Duration,
	logger *logrus.Logger,
) (*SocketBabbleProxy, error) {

	if logger == nil {
		logger = logrus.New()

		logger.Level = logrus.DebugLevel
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
