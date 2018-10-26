package app

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/sirupsen/logrus"
)

type SocketAppProxyClient struct {
	clientAddr string
	timeout    time.Duration
	logger     *logrus.Logger
	rpc        *rpc.Client
}

func NewSocketAppProxyClient(clientAddr string, timeout time.Duration, logger *logrus.Logger) *SocketAppProxyClient {
	return &SocketAppProxyClient{
		clientAddr: clientAddr,
		timeout:    timeout,
		logger:     logger,
	}
}

func (p *SocketAppProxyClient) getConnection() error {
	if p.rpc == nil {
		conn, err := net.DialTimeout("tcp", p.clientAddr, p.timeout)

		if err != nil {
			return err
		}

		p.rpc = jsonrpc.NewClient(conn)
	}

	return nil
}

func (p *SocketAppProxyClient) CommitBlock(block hashgraph.Block) ([]byte, error) {
	if err := p.getConnection(); err != nil {
		return []byte{}, err
	}

	var stateHash []byte

	if err := p.rpc.Call("State.CommitBlock", block, &stateHash); err != nil {
		p.rpc = nil

		return []byte{}, err
	}

	p.logger.WithFields(logrus.Fields{
		"block":      block.Index(),
		"state_hash": stateHash,
	}).Debug("AppProxyClient.CommitBlock")

	return stateHash, nil
}

func (p *SocketAppProxyClient) GetSnapshot(blockIndex int) ([]byte, error) {
	if err := p.getConnection(); err != nil {
		return []byte{}, err
	}

	var snapshot []byte

	if err := p.rpc.Call("State.GetSnapshot", blockIndex, &snapshot); err != nil {
		p.rpc = nil

		return []byte{}, err
	}

	p.logger.WithFields(logrus.Fields{
		"block":    blockIndex,
		"snapshot": snapshot,
	}).Debug("AppProxyClient.GetSnapshot")

	return snapshot, nil
}

func (p *SocketAppProxyClient) Restore(snapshot []byte) error {
	if err := p.getConnection(); err != nil {
		return err
	}

	var stateHash []byte

	if err := p.rpc.Call("State.Restore", snapshot, &stateHash); err != nil {
		p.rpc = nil

		return err
	}

	p.logger.WithFields(logrus.Fields{
		"state_hash": stateHash,
	}).Debug("AppProxyClient.Restore")

	return nil
}
