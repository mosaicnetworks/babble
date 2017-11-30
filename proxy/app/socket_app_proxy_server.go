package app

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/sirupsen/logrus"
)

type SocketAppProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	submitCh    chan []byte
	logger      *logrus.Logger
}

func NewSocketAppProxyServer(bindAddress string, logger *logrus.Logger) *SocketAppProxyServer {
	server := &SocketAppProxyServer{
		submitCh: make(chan []byte),
		logger:   logger,
	}
	server.register(bindAddress)
	return server
}

func (p *SocketAppProxyServer) register(bindAddress string) {
	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("Babble", p)
	p.rpcServer = rpcServer

	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		p.logger.WithField("error", err).Error("Failed to listen")
	}
	p.netListener = &l
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

func (p *SocketAppProxyServer) SubmitTx(tx []byte, ack *bool) error {
	p.logger.Debug("SubmitTx")
	p.submitCh <- tx
	*ack = true
	return nil
}
