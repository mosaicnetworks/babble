package babble

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

// SocketBabbleProxyClient is the client component of the BabbleProxy that sends
// RPC requests to Babble
type SocketBabbleProxyClient struct {
	nodeAddr string
	timeout  time.Duration
	rpc      *rpc.Client
}

// NewSocketBabbleProxyClient implements a new SocketBabbleProxyClient
func NewSocketBabbleProxyClient(nodeAddr string, timeout time.Duration) *SocketBabbleProxyClient {
	return &SocketBabbleProxyClient{
		nodeAddr: nodeAddr,
		timeout:  timeout,
	}
}

func (p *SocketBabbleProxyClient) getConnection() error {
	if p.rpc == nil {
		conn, err := net.DialTimeout("tcp", p.nodeAddr, p.timeout)

		if err != nil {
			return err
		}

		p.rpc = jsonrpc.NewClient(conn)
	}

	return nil
}

// SubmitTx submits a transaction to Babble
func (p *SocketBabbleProxyClient) SubmitTx(tx []byte) (*bool, error) {
	if err := p.getConnection(); err != nil {
		return nil, err
	}

	var ack bool

	err := p.rpc.Call("Babble.SubmitTx", tx, &ack)

	if err != nil {
		p.rpc = nil

		return nil, err
	}

	return &ack, nil
}
