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
	retries  int
}

// NewSocketBabbleProxyClient implements a new SocketBabbleProxyClient
func NewSocketBabbleProxyClient(nodeAddr string, timeout time.Duration) *SocketBabbleProxyClient {
	return &SocketBabbleProxyClient{
		nodeAddr: nodeAddr,
		timeout:  timeout,
		retries:  3,
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

// call is a wrapper around rpc.Call, that implements a retry mechanism
func (p *SocketBabbleProxyClient) call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error
	for try := 0; try < p.retries; try++ {
		err = p.getConnection()
		if err != nil {
			// this attempt failed; try again
			continue
		}
		err = p.rpc.Call(serviceMethod, args, reply)
		if err != nil {
			// this attempt failed; reset connection, log, and try again
			p.rpc = nil
			continue
		}
		// attempt succeeded, return
		break
	}
	return err
}

// SubmitTx submits a transaction to Babble
func (p *SocketBabbleProxyClient) SubmitTx(tx []byte) (*bool, error) {
	var ack bool
	err := p.call("Babble.SubmitTx", tx, &ack)
	if err != nil {
		return nil, err
	}
	return &ack, nil
}
