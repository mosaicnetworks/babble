package app

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/sirupsen/logrus"
)

type SocketAppProxyServer struct {
	netListener      *net.Listener
	rpcServer        *rpc.Server
	submitCh         chan []byte
	submitInternalCh chan hashgraph.InternalTransaction
	logger           *logrus.Logger
}

func NewSocketAppProxyServer(bindAddress string, logger *logrus.Logger) (*SocketAppProxyServer, error) {
	server := &SocketAppProxyServer{
		submitCh:         make(chan []byte),
		submitInternalCh: make(chan hashgraph.InternalTransaction),
		logger:           logger,
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

func (p *SocketAppProxyServer) SubmitTx(tx []byte, ack *bool) error {
	p.logger.Debug("SubmitTx")

	p.submitCh <- tx

	*ack = true

	return nil
}

func (p *SocketAppProxyServer) SubmitInternalTx(tx hashgraph.InternalTransaction, ack *bool) error {
	p.logger.Debug("SubmitTx")

	p.submitInternalCh <- tx

	*ack = true

	return nil
}
