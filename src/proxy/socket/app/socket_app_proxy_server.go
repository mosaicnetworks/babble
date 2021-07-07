package app

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/sirupsen/logrus"
)

// SocketAppProxyServer is the server component of the AppProxy that responds
// to RPC requests from App.
type SocketAppProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	submitCh    chan []byte
	logger      *logrus.Entry
}

// NewSocketAppProxyServer creates a new SocketAppProxyServer
func NewSocketAppProxyServer(bindAddress string, logger *logrus.Entry) (*SocketAppProxyServer, error) {
	server := &SocketAppProxyServer{
		submitCh: make(chan []byte),
		logger:   logger,
	}

	if err := server.register(bindAddress); err != nil {
		return nil, err
	}

	return server, nil
}

func (p *SocketAppProxyServer) register(bindAddress string) error {
	rpcServer := rpc.NewServer()

	rpcServer.RegisterName("Babble", p)

	p.rpcServer = rpcServer

	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		p.logger.WithField("error", err).Error("Failed to listen")
		return err
	}

	p.netListener = &l

	return nil
}

func (p *SocketAppProxyServer) listen() {
	for {
		conn, err := (*p.netListener).Accept()
		if err != nil {
			p.logger.WithField("error", err).Error("Failed to accept")
		}

		go (*p.rpcServer).ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

// SubmitTx Implements the AppProxy interface
func (p *SocketAppProxyServer) SubmitTx(tx []byte, ack *bool) error {
	p.logger.Debug("SubmitTx")

	p.submitCh <- tx

	*ack = true

	return nil
}
