package app

import (
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

type SocketAppProxy struct {
	clientAddress string
	bindAddress   string

	client *SocketAppProxyClient
	server *SocketAppProxyServer

	logger *logrus.Logger
}

func NewSocketAppProxy(clientAddr string, bindAddr string, timeout time.Duration, logger *logrus.Logger) (*SocketAppProxy, error) {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	client := NewSocketAppProxyClient(clientAddr, timeout, logger)

	server, err := NewSocketAppProxyServer(bindAddr, logger)

	if err != nil {
		return nil, err
	}

	proxy := &SocketAppProxy{
		clientAddress: clientAddr,
		bindAddress:   bindAddr,
		client:        client,
		server:        server,
		logger:        logger,
	}

	go proxy.server.listen()

	return proxy, nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Implement AppProxy Interface

func (p *SocketAppProxy) SubmitCh() chan []byte {
	return p.server.submitCh
}

func (p *SocketAppProxy) SubmitInternalCh() chan hashgraph.InternalTransaction {
	return p.server.submitInternalCh
}

func (p *SocketAppProxy) CommitBlock(block hashgraph.Block) (proxy.CommitResponse, error) {
	return p.client.CommitBlock(block)
}

func (p *SocketAppProxy) GetSnapshot(blockIndex int) ([]byte, error) {
	return p.client.GetSnapshot(blockIndex)
}

func (p *SocketAppProxy) Restore(snapshot []byte) error {
	return p.client.Restore(snapshot)
}
