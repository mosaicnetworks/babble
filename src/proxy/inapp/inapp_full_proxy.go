package app

import (
	"fmt"
	"time"

	bcrypto "github.com/mosaicnetworks/babble/src/crypto"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	bproxy "github.com/mosaicnetworks/babble/src/proxy/babble"
	"github.com/sirupsen/logrus"
)

//InmemFullProxy serves as an Inapp proxy for those whom
type InmemFullProxy struct {
	submitCh              chan []byte
	commitCh              chan bproxy.Commit
	stateHash             []byte
	committedTransactions [][]byte
	logger                *logrus.Logger
	snapshotRequestCh     chan bproxy.SnapshotRequest
	restoreCh             chan bproxy.RestoreRequest
	timeout               time.Duration
}

// NewInmemFullProxy instantiate an InmemFullProxy.
// If no logger, a new one is created
func NewInmemFullProxy(timeout time.Duration, logger *logrus.Logger) *InmemFullProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	return &InmemFullProxy{
		submitCh:              make(chan []byte),
		commitCh:              make(chan bproxy.Commit),
		stateHash:             []byte{},
		committedTransactions: [][]byte{},
		logger:                logger,
		snapshotRequestCh:     make(chan bproxy.SnapshotRequest),
		restoreCh:             make(chan bproxy.RestoreRequest),
		timeout:               timeout,
	}
}

func (p *InmemFullProxy) commit(block hg.Block) []byte {
	p.committedTransactions = append(p.committedTransactions, block.Transactions()...)

	hash := p.stateHash

	for _, t := range block.Transactions() {
		tHash := bcrypto.SHA256(t)

		hash = bcrypto.SimpleHashFromTwoHashes(hash, tHash)
	}

	p.stateHash = hash

	return p.stateHash
}

//------------------------------------------------------------------------------
//Implement AppProxy Interface

// SubmitCh returns the channel of raw transactions
func (p *InmemFullProxy) SubmitCh() chan []byte {
	return p.submitCh
}

// CommitBlock send the block to the user and wait for an answer.
// It update the state hash accordingly
func (p *InmemFullProxy) CommitBlock(block hg.Block) (res []byte, err error) {
	respCh := make(chan bproxy.CommitResponse)

	p.commitCh <- bproxy.Commit{
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
func (p *InmemFullProxy) GetSnapshot(blockIndex int) (res []byte, err error) {
	respCh := make(chan bproxy.SnapshotResponse)

	p.snapshotRequestCh <- bproxy.SnapshotRequest{
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

func (p *InmemFullProxy) Restore(snapshot []byte) (err error) {
	// Send the Request over
	respCh := make(chan bproxy.RestoreResponse)

	p.restoreCh <- bproxy.RestoreRequest{
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
func (p *InmemFullProxy) SubmitTx(tx []byte) error {
	p.submitCh <- tx

	return nil
}

// CommitCh returns the channel of blocks comming from the network
func (p *InmemFullProxy) CommitCh() chan bproxy.Commit {
	return p.commitCh
}

// SnapshotRequestCh returns the channel of incoming SnapshotRequest
func (p *InmemFullProxy) SnapshotRequestCh() chan bproxy.SnapshotRequest {
	return p.snapshotRequestCh
}

// RestoreCh returns the channel of incoming RestoreRequest
func (p *InmemFullProxy) RestoreCh() chan bproxy.RestoreRequest {
	return p.restoreCh
}
