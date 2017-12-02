package app

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

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

func (p *SocketAppProxyClient) CommitTx(tx []byte) (*bool, error) {
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
