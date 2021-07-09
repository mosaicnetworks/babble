package app

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

const DefaultProxyTimeout = time.Second

// SocketAppProxyClient is the component of the AppProxy that sends RPC requests
// to the App
type SocketAppProxyClient struct {
	clientAddr string
	timeout    time.Duration
	logger     *logrus.Entry
	rpc        *rpc.Client
	retries    int
}

// NewSocketAppProxyClient creates a new SocketAppProxyClient
func NewSocketAppProxyClient(clientAddr string, timeout time.Duration, logger *logrus.Entry) *SocketAppProxyClient {
	return &SocketAppProxyClient{
		clientAddr: clientAddr,
		timeout:    timeout,
		logger:     logger,
		retries:    3,
	}
}

func (p *SocketAppProxyClient) getConnection() error {
	if p.rpc == nil {
		conn, err := net.DialTimeout("tcp", p.clientAddr, p.timeout)
		if err != nil {
			return err
		}
		p.rpc = jsonrpc.NewClient(conn)
	}
	return nil
}

// call is a wrapper around rpc.Call, that implements a retry mechanism
func (p *SocketAppProxyClient) call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error
	for try := 0; try < p.retries; try++ {
		err = p.getConnection()
		if err != nil {
			// this attempt failed; log and try again
			p.logger.Debugf("proxy_client getConnection failed (%d of %d): %v", try+1, p.retries, err)
			continue
		}

		call := p.rpc.Go(serviceMethod, args, reply, nil)
		select {
		case <-time.After(p.timeout):
			err = fmt.Errorf("rpc timeout")
			break
		case <-call.Done:
			err = call.Error
			break
		}

		if err != nil {
			// this attempt failed; reset connection, log, and try again
			p.rpc.Close()
			p.rpc = nil
			p.logger.Debugf("proxy_client call failed (%d of %d): %v", try+1, p.retries, err)
			continue
		}

		// attempt succeeded, return
		break
	}
	return err
}

// CommitBlock implements the AppProxy interface
func (p *SocketAppProxyClient) CommitBlock(block hashgraph.Block) (proxy.CommitResponse, error) {
	var commitResponse proxy.CommitResponse
	if err := p.call("State.CommitBlock", block, &commitResponse); err != nil {
		return commitResponse, err
	}

	jsonResp, _ := json.Marshal(commitResponse)
	p.logger.WithFields(logrus.Fields{
		"block":           block.Index(),
		"commit_response": string(jsonResp),
	}).Debug("AppProxyClient.CommitBlock")

	return commitResponse, nil
}

// GetSnapshot implementes the AppProxy interface
func (p *SocketAppProxyClient) GetSnapshot(blockIndex int) ([]byte, error) {
	var snapshot []byte
	if err := p.call("State.GetSnapshot", blockIndex, &snapshot); err != nil {
		return []byte{}, err
	}

	p.logger.WithFields(logrus.Fields{
		"block":    blockIndex,
		"snapshot": snapshot,
	}).Debug("AppProxyClient.GetSnapshot")

	return snapshot, nil
}

// Restore implements the AppProxy interface
func (p *SocketAppProxyClient) Restore(snapshot []byte) error {
	var stateHash []byte
	if err := p.call("State.Restore", snapshot, &stateHash); err != nil {
		return err
	}

	p.logger.WithFields(logrus.Fields{
		"state_hash": stateHash,
	}).Debug("AppProxyClient.Restore")

	return nil
}

// OnStateChanged implements the AppProxy interface
func (p *SocketAppProxyClient) OnStateChanged(state state.State) error {
	if err := p.call("State.OnStateChanged", state, nil); err != nil {
		return err
	}

	p.logger.WithFields(logrus.Fields{
		"state": state.String,
	}).Debug("AppProxyClient.OnStateChanged")

	return nil
}
