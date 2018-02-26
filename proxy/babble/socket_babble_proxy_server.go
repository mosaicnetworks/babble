package babble

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/babbleio/babble/hashgraph"
)

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

type SocketBabbleProxyServer struct {
	netListener *net.Listener
	rpcServer   *rpc.Server
	commitCh    chan Commit
	timeout     time.Duration
}

func NewSocketBabbleProxyServer(bindAddress string, timeout time.Duration) (*SocketBabbleProxyServer, error) {
	server := &SocketBabbleProxyServer{
		commitCh: make(chan Commit),
		timeout:  timeout,
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

func (p *SocketBabbleProxyServer) CommitBlock(block hashgraph.Block, stateHash *[]byte) (err error) {
	// Send the Commit over
	respCh := make(chan CommitResponse)
	p.commitCh <- Commit{
		Block:    block,
		RespChan: respCh,
	}

	// Wait for a response
	select {
	case commitResp := <-respCh:
		stateHash = &commitResp.StateHash
		if commitResp.Error != nil {
			err = commitResp.Error
		}
	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}
	return

}
