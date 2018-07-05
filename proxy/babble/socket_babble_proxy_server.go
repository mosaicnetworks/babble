package babble

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mosaicnetworks/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

//------------------------------------------------------------------------------

type StateHash struct {
	Hash []byte
}

// CommitResponse captures both a response and a potential error.
type CommitResponse struct {
	StateHash []byte
	Error     error
}

// Commit provides a response mechanism.
type Commit struct {
	Block    hashgraph.Block
	RespChan chan<- CommitResponse
}

// Respond is used to respond with a response, error or both
func (r *Commit) Respond(stateHash []byte, err error) {
	r.RespChan <- CommitResponse{stateHash, err}
}

//------------------------------------------------------------------------------

type Snapshot struct {
	Bytes []byte
}

// SnapshotResponse captures both a response and a potential error.
type SnapshotResponse struct {
	Snapshot []byte
	Error    error
}

// SnapshotRequest provides a response mechanism.
type SnapshotRequest struct {
	BlockIndex int
	RespChan   chan<- SnapshotResponse
}

// Respond is used to respond with a response, error or both
func (r *SnapshotRequest) Respond(snapshot []byte, err error) {
	r.RespChan <- SnapshotResponse{snapshot, err}
}

//------------------------------------------------------------------------------

// RestoreResponse captures both an error.
type RestoreResponse struct {
	StateHash []byte
	Error     error
}

// RestoreRequest provides a response mechanism.
type RestoreRequest struct {
	Snapshot []byte
	RespChan chan<- RestoreResponse
}

// Respond is used to respond with a response, error or both
func (r *RestoreRequest) Respond(snapshot []byte, err error) {
	r.RespChan <- RestoreResponse{snapshot, err}
}

//------------------------------------------------------------------------------

type SocketBabbleProxyServer struct {
	netListener       *net.Listener
	rpcServer         *rpc.Server
	commitCh          chan Commit
	snapshotRequestCh chan SnapshotRequest
	restoreCh         chan RestoreRequest
	timeout           time.Duration
	logger            *logrus.Logger
}

func NewSocketBabbleProxyServer(bindAddress string,
	timeout time.Duration,
	logger *logrus.Logger) (*SocketBabbleProxyServer, error) {

	server := &SocketBabbleProxyServer{
		commitCh:          make(chan Commit),
		snapshotRequestCh: make(chan SnapshotRequest),
		restoreCh:         make(chan RestoreRequest),
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

func (p *SocketBabbleProxyServer) CommitBlock(block hashgraph.Block, stateHash *StateHash) (err error) {
	// Send the Commit over
	respCh := make(chan CommitResponse)
	p.commitCh <- Commit{
		Block:    block,
		RespChan: respCh,
	}

	// Wait for a response
	select {
	case commitResp := <-respCh:
		stateHash.Hash = commitResp.StateHash
		if commitResp.Error != nil {
			err = commitResp.Error
		}
	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"block":      block.Index(),
		"state_hash": stateHash.Hash,
		"err":        err,
	}).Debug("BabbleProxyServer.CommitBlock")

	return

}

func (p *SocketBabbleProxyServer) GetSnapshot(blockIndex int, snapshot *Snapshot) (err error) {
	// Send the Request over
	respCh := make(chan SnapshotResponse)
	p.snapshotRequestCh <- SnapshotRequest{
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

func (p *SocketBabbleProxyServer) Restore(snapshot []byte, stateHash *StateHash) (err error) {
	// Send the Request over
	respCh := make(chan RestoreResponse)
	p.restoreCh <- RestoreRequest{
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
