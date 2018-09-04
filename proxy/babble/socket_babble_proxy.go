package babble

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type SocketBabbleProxy struct {
	nodeAddress string
	bindAddress string

	client *SocketBabbleProxyClient
	server *SocketBabbleProxyServer
}

func NewSocketBabbleProxy(nodeAddr string,
	bindAddr string,
	timeout time.Duration,
	logger *logrus.Logger) (*SocketBabbleProxy, error) {

	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	client := NewSocketBabbleProxyClient(nodeAddr, timeout)
	server, err := NewSocketBabbleProxyServer(bindAddr, timeout, logger)
	if err != nil {
		return nil, err
	}

	proxy := &SocketBabbleProxy{
		nodeAddress: nodeAddr,
		bindAddress: bindAddr,
		client:      client,
		server:      server,
	}
	go proxy.server.listen()

	return proxy, nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Implement BabbleProxy interface

func (p *SocketBabbleProxy) CommitCh() chan Commit {
	return p.server.commitCh
}

func (p *SocketBabbleProxy) SnapshotRequestCh() chan SnapshotRequest {
	return p.server.snapshotRequestCh
}

func (p *SocketBabbleProxy) RestoreCh() chan RestoreRequest {
	return p.server.restoreCh
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
