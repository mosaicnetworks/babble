package app

import (
	"time"

	"github.com/mosaicnetworks/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

type SocketAppProxy struct {
	clientAddress string
	bindAddress   string

	client *SocketAppProxyClient
	server *SocketAppProxyServer

	logger *logrus.Logger
}

func NewSocketAppProxy(clientAddr string, bindAddr string, timeout time.Duration, logger *logrus.Logger) *SocketAppProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	client := NewSocketAppProxyClient(clientAddr, timeout, logger)
	server := NewSocketAppProxyServer(bindAddr, logger)

	proxy := &SocketAppProxy{
		clientAddress: clientAddr,
		bindAddress:   bindAddr,
		client:        client,
		server:        server,
		logger:        logger,
	}
	go proxy.server.listen()

	return proxy
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Implement AppProxy Interface

func (p *SocketAppProxy) SubmitCh() chan []byte {
	return p.server.submitCh
}

func (p *SocketAppProxy) CommitBlock(block hashgraph.Block) ([]byte, error) {
	return p.client.CommitBlock(block)
}

func (p *SocketAppProxy) GetSnapshot(blockIndex int) ([]byte, error) {
	return p.client.GetSnapshot(blockIndex)
}

func (p *SocketAppProxy) Restore(snapshot []byte) error {
	return p.client.Restore(snapshot)
}
