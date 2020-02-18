package babble

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

// SocketBabbleProxyServer is the server component of the BabbleProxy which
// responds to RPC requests from the client component of the AppProxy
type SocketBabbleProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	handler     proxy.ProxyHandler
	timeout     time.Duration
	logger      *logrus.Entry
}

// NewSocketBabbleProxyServer creates a new SocketBabbleProxyServer
func NewSocketBabbleProxyServer(
	bindAddress string,
	handler proxy.ProxyHandler,
	timeout time.Duration,
	logger *logrus.Entry,
) (*SocketBabbleProxyServer, error) {

	server := &SocketBabbleProxyServer{
		handler: handler,
		timeout: timeout,
		logger:  logger,
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
}

// CommitBlock implements the AppProxy interface
func (p *SocketBabbleProxyServer) CommitBlock(block hashgraph.Block, response *proxy.CommitResponse) (err error) {
	*response, err = p.handler.CommitHandler(block)

	p.logger.WithFields(logrus.Fields{
		"block":    block.Index(),
		"response": response,
		"err":      err,
	}).Debug("BabbleProxyServer.CommitBlock")

	return
}

// GetSnapshot implements the AppProxy interface
func (p *SocketBabbleProxyServer) GetSnapshot(blockIndex int, snapshot *[]byte) (err error) {
	*snapshot, err = p.handler.SnapshotHandler(blockIndex)

	p.logger.WithFields(logrus.Fields{
		"block":    blockIndex,
		"snapshot": snapshot,
		"err":      err,
	}).Debug("BabbleProxyServer.GetSnapshot")

	return
}

// Restore implements the AppProxy interface
func (p *SocketBabbleProxyServer) Restore(snapshot []byte, stateHash *[]byte) (err error) {
	*stateHash, err = p.handler.RestoreHandler(snapshot)

	p.logger.WithFields(logrus.Fields{
		"state_hash": stateHash,
		"err":        err,
	}).Debug("BabbleProxyServer.Restore")

	return
}

// OnStateChanged implements the AppProxy interface
func (p *SocketBabbleProxyServer) OnStateChanged(state state.State, obj *struct{}) (err error) {
	err = p.handler.StateChangeHandler(state)

	p.logger.WithFields(logrus.Fields{
		"state": state.String(),
		"err":   err,
	}).Debug("BabbleProxyServer.OnStateChanged")

	return
}
