package babble

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/babbleio/babble/hashgraph"
)

type SocketBabbleProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	commitCh    chan hashgraph.Block
}

func NewSocketBabbleProxyServer(bindAddress string) (*SocketBabbleProxyServer, error) {
	server := &SocketBabbleProxyServer{
		commitCh: make(chan hashgraph.Block),
	}

	if err := server.register(bindAddress); err != nil {
		return nil, err
	}

	return server, nil
}

func (p *SocketBabbleProxyServer) register(bindAddress string) error {
	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("State", p)
	p.rpcServer = rpcServer

	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		return err
	}

	p.netListener = &l

	return nil
}

func (p *SocketBabbleProxyServer) listen() error {
	for {
		conn, err := (*p.netListener).Accept()
		if err != nil {
			return err
		}

		go (*p.rpcServer).ServeCodec(jsonrpc.NewServerCodec(conn))
	}
	return nil
}

func (p *SocketBabbleProxyServer) CommitBlock(block hashgraph.Block, ack *bool) error {
	p.commitCh <- block
	*ack = true
	return nil
}
