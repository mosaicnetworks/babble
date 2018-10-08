package babble

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy/proto"
	"github.com/sirupsen/logrus"
)

type SocketBabbleProxyServer struct {
	netListener       *net.Listener
	rpcServer         *rpc.Server
	commitCh          chan proto.Commit
	snapshotRequestCh chan proto.SnapshotRequest
	restoreCh         chan proto.RestoreRequest
	timeout           time.Duration
	logger            *logrus.Logger
}

func NewSocketBabbleProxyServer(bindAddress string,
	timeout time.Duration,
	logger *logrus.Logger) (*SocketBabbleProxyServer, error) {

	server := &SocketBabbleProxyServer{
		commitCh:          make(chan proto.Commit),
		snapshotRequestCh: make(chan proto.SnapshotRequest),
		restoreCh:         make(chan proto.RestoreRequest),
		timeout:           timeout,
		logger:            logger,
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

func (p *SocketBabbleProxyServer) CommitBlock(block hashgraph.Block, stateH *proto.StateHash) (err error) {
	// Send the Commit over
	respCh := make(chan proto.CommitResponse)

	p.commitCh <- proto.Commit{
		Block:    block,
		RespChan: respCh,
	}

	// Wait for a response
	select {
	case commitResp := <-respCh:
		stateH.Hash = commitResp.StateHash

		if commitResp.Error != nil {
			err = commitResp.Error
		}

	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"block":      block.Index(),
		"state_hash": stateH.Hash,
		"err":        err,
	}).Debug("BabbleProxyServer.CommitBlock")

	return
}

func (p *SocketBabbleProxyServer) GetSnapshot(blockIndex int, snapshot *proto.Snapshot) (err error) {
	// Send the Request over
	respCh := make(chan proto.SnapshotResponse)

	p.snapshotRequestCh <- proto.SnapshotRequest{
		BlockIndex: blockIndex,
		RespChan:   respCh,
	}

	// Wait for a response
	select {
	case snapshotResp := <-respCh:
		snapshot.Bytes = snapshotResp.Snapshot

		if snapshotResp.Error != nil {
			err = snapshotResp.Error
		}

	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"block":    blockIndex,
		"snapshot": snapshot.Bytes,
		"err":      err,
	}).Debug("BabbleProxyServer.GetSnapshot")

	return
}

func (p *SocketBabbleProxyServer) Restore(snapshot []byte, stateHash *proto.StateHash) (err error) {
	// Send the Request over
	respCh := make(chan proto.RestoreResponse)

	p.restoreCh <- proto.RestoreRequest{
		Snapshot: snapshot,
		RespChan: respCh,
	}

	// Wait for a response
	select {
	case restoreResp := <-respCh:
		stateHash.Hash = restoreResp.StateHash

		if restoreResp.Error != nil {
			err = restoreResp.Error
		}

	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"state_hash": stateHash.Hash,
		"err":        err,
	}).Debug("BabbleProxyServer.Restore")

	return
}
