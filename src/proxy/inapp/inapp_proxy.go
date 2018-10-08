package inapp

import (
	"fmt"
	"time"

	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy/proto"
	"github.com/sirupsen/logrus"
)

//InmemProxy serves as an Inapp proxy for those whom
type InappProxy struct {
	submitCh              chan []byte
	commitCh              chan proto.Commit
	stateHash             []byte
	committedTransactions [][]byte
	logger                *logrus.Logger
	snapshotRequestCh     chan proto.SnapshotRequest
	restoreCh             chan proto.RestoreRequest
	timeout               time.Duration
}

// NewInappProxy instantiate an InappProxy.
// If no logger, a new one is created
func NewInappProxy(timeout time.Duration, logger *logrus.Logger) *InappProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	return &InappProxy{
		submitCh:              make(chan []byte),
		commitCh:              make(chan proto.Commit),
		stateHash:             []byte{},
		committedTransactions: [][]byte{},
		logger:                logger,
		snapshotRequestCh:     make(chan proto.SnapshotRequest),
		restoreCh:             make(chan proto.RestoreRequest),
		timeout:               timeout,
	}
}

//------------------------------------------------------------------------------
//Implement AppProxy Interface

// SubmitCh returns the channel of raw transactions
func (p *InappProxy) SubmitCh() chan []byte {
	return p.submitCh
}

// CommitBlock send the block to the user and wait for an answer.
// It update the state hash accordingly
func (p *InappProxy) CommitBlock(block hg.Block) (res []byte, err error) {
	respCh := make(chan proto.CommitResponse)

	p.commitCh <- proto.Commit{
		Block:    block,
		RespChan: respCh,
	}

	// Wait for a response
	select {
	case commitResp := <-respCh:
		res = commitResp.StateHash

		if commitResp.Error != nil {
			err = commitResp.Error
		}

	case <-time.After(p.timeout): // Fixme
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"round_received": block.RoundReceived(),
		"txs":            len(block.Transactions()),
		"hash":           res,
		"err":            err,
	}).Debug("InmemProxy CommitBlock")

	return
}

//TODO - Implement these two functions
func (p *InappProxy) GetSnapshot(blockIndex int) (res []byte, err error) {
	respCh := make(chan proto.SnapshotResponse)

	p.snapshotRequestCh <- proto.SnapshotRequest{
		BlockIndex: blockIndex,
		RespChan:   respCh,
	}

	// Wait for a response
	select {
	case snapshotResp := <-respCh:
		res = snapshotResp.Snapshot

		if snapshotResp.Error != nil {
			err = snapshotResp.Error
		}

	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"block":    blockIndex,
		"snapshot": res,
		"err":      err,
	}).Debug("InmemProxy.GetSnapshot")

	return
}

func (p *InappProxy) Restore(snapshot []byte) (err error) {
	// Send the Request over
	respCh := make(chan proto.RestoreResponse)

	p.restoreCh <- proto.RestoreRequest{
		Snapshot: snapshot,
		RespChan: respCh,
	}

	var stateHash []byte

	// Wait for a response
	select {
	case restoreResp := <-respCh:
		stateHash = restoreResp.StateHash

		if restoreResp.Error != nil {
			err = restoreResp.Error
		}

	case <-time.After(p.timeout):
		err = fmt.Errorf("command timed out")
	}

	p.logger.WithFields(logrus.Fields{
		"state_hash": stateHash,
		"err":        err,
	}).Debug("BabbleProxyServer.Restore")

	return
}

//------------------------------------------------------------------------------
//Implement BabbleProxy Interface

// SubmitTx adds a transaction to be validated
func (p *InappProxy) SubmitTx(tx []byte) error {
	p.submitCh <- tx

	return nil
}

// CommitCh returns the channel of blocks comming from the network
func (p *InappProxy) CommitCh() chan proto.Commit {
	return p.commitCh
}

// SnapshotRequestCh returns the channel of incoming SnapshotRequest
func (p *InappProxy) SnapshotRequestCh() chan proto.SnapshotRequest {
	return p.snapshotRequestCh
}

// RestoreCh returns the channel of incoming RestoreRequest
func (p *InappProxy) RestoreCh() chan proto.RestoreRequest {
	return p.restoreCh
}
