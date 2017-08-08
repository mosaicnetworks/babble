package babble

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type SocketBabbleProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	commitCh    chan []byte
}

func NewSocketBabbleProxyServer(bindAddress string) (*SocketBabbleProxyServer, error) {
	server := &SocketBabbleProxyServer{
		commitCh: make(chan []byte),
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

func (p *SocketBabbleProxyServer) CommitTx(tx []byte, ack *bool) error {
	p.commitCh <- tx
	*ack = true
	return nil
}
