package babble

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

type SocketBabbleProxyClient struct {
	nodeAddr string
	timeout  time.Duration
}

func NewSocketBabbleProxyClient(nodeAddr string, timeout time.Duration) *SocketBabbleProxyClient {
	return &SocketBabbleProxyClient{
		nodeAddr: nodeAddr,
		timeout:  timeout,
	}
}

func (p *SocketBabbleProxyClient) getConnection() (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", p.nodeAddr, p.timeout)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

func (p *SocketBabbleProxyClient) SubmitTx(tx []byte) (*bool, error) {
	rpcConn, err := p.getConnection()
	if err != nil {
		return nil, err
	}
	var ack bool
	err = rpcConn.Call("Babble.SubmitTx", tx, &ack)
	if err != nil {
		return nil, err
	}
	return &ack, nil
}
