package babble

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/peers"
)

type SocketBabbleProxyClient struct {
	nodeAddr string
	timeout  time.Duration
	rpc      *rpc.Client
}

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

func (p *SocketBabbleProxyClient) ProposePeerAdd(peer peers.Peer) (*bool, error) {
	if err := p.getConnection(); err != nil {
		return nil, err
	}

	var ack bool

	err := p.rpc.Call("Babble.ProposePeerAdd", []hashgraph.InternalTransaction{hashgraph.NewInternalTransaction(hashgraph.PEER_ADD, peer)}, &ack)

	if err != nil {
		p.rpc = nil

		return nil, err
	}

	return &ack, nil
}

func (p *SocketBabbleProxyClient) ProposePeerRemove(peer peers.Peer) (*bool, error) {
	if err := p.getConnection(); err != nil {
		return nil, err
	}

	var ack bool

	err := p.rpc.Call("Babble.ProposePeerAdd", []hashgraph.InternalTransaction{hashgraph.NewInternalTransaction(hashgraph.PEER_REMOVE, peer)}, &ack)

	if err != nil {
		p.rpc = nil

		return nil, err
	}

	return &ack, nil
}
