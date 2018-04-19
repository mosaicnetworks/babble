package app

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/babbleio/babble/hashgraph"
	bp "github.com/babbleio/babble/proxy/babble"
	"github.com/sirupsen/logrus"
)

type SocketAppProxyClient struct {
	clientAddr string
	timeout    time.Duration
	logger     *logrus.Logger
}

func NewSocketAppProxyClient(clientAddr string, timeout time.Duration, logger *logrus.Logger) *SocketAppProxyClient {
	return &SocketAppProxyClient{
		clientAddr: clientAddr,
		timeout:    timeout,
		logger:     logger,
	}
}

func (p *SocketAppProxyClient) getConnection() (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", p.clientAddr, p.timeout)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

func (p *SocketAppProxyClient) CommitBlock(block hashgraph.Block) ([]byte, error) {
	rpcConn, err := p.getConnection()
	if err != nil {
		return nil, err
	}

	var stateHash bp.StateHash
	err = rpcConn.Call("State.CommitBlock", block, &stateHash)

	p.logger.WithFields(logrus.Fields{
		"block":      block.Index(),
		"state_hash": stateHash.Hash,
	}).Debug("AppProxyClient.CommitBlock")

	return stateHash.Hash, err
}
