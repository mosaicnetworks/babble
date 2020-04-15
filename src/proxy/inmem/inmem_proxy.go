// Package inmem implements an in-memory AppProxy to use Babble directly from Go
// code.
package inmem

import (
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

// InmemProxy implements the AppProxy interface natively. It requires a
// ProxyHandler that implements the callbacks that will be called to update the
// application.
type InmemProxy struct {
	handler  proxy.ProxyHandler
	submitCh chan []byte
	logger   *logrus.Entry
}

// NewInmemProxy instantiates an InmemProxy from a set of handlers. If logger is
// nil, a new one is created.
func NewInmemProxy(handler proxy.ProxyHandler,
	logger *logrus.Entry) *InmemProxy {

	if logger == nil {
		log := logrus.New()
		log.Level = logrus.DebugLevel
		logger = logrus.NewEntry(log)
	}

	return &InmemProxy{
		handler:  handler,
		submitCh: make(chan []byte),
		logger:   logger,
	}
}

/*******************************************************************************
* SubmitTx                                                                     *
*******************************************************************************/

// SubmitTx is called by the App to submit a transaction to Babble.
func (p *InmemProxy) SubmitTx(tx []byte) {
	//have to make a copy, or the tx will be garbage collected and weird stuff
	//happens in transaction pool
	t := make([]byte, len(tx), len(tx))

	copy(t, tx)

	p.submitCh <- t
}

/*******************************************************************************
* Implement AppProxy Interface                                                 *
*******************************************************************************/

// SubmitCh is used internally by Babble to retrieve the channel through which
// transactions are received from the App.
func (p *InmemProxy) SubmitCh() chan []byte {
	return p.submitCh
}

// CommitBlock calls the CommitHandler.
func (p *InmemProxy) CommitBlock(block hg.Block) (proxy.CommitResponse, error) {
	commitResponse, err := p.handler.CommitHandler(block)

	if p.logger.Level > logrus.InfoLevel {
		blockBytes, _ := block.Marshal()
		p.logger.WithFields(logrus.Fields{
			"block":    string(blockBytes),
			"txs":      len(block.Transactions()),
			"response": commitResponse,
			"err":      err,
		}).Debug("InmemProxy.CommitBlock")

	}

	return commitResponse, err
}

// GetSnapshot calls the SnapshotHandler.
func (p *InmemProxy) GetSnapshot(blockIndex int) ([]byte, error) {
	snapshot, err := p.handler.SnapshotHandler(blockIndex)

	p.logger.WithFields(logrus.Fields{
		"block":    blockIndex,
		"snapshot": snapshot,
		"err":      err,
	}).Debug("InmemProxy.GetSnapshot")

	return snapshot, err
}

// Restore calls the RestoreHandler.
func (p *InmemProxy) Restore(snapshot []byte) error {
	stateHash, err := p.handler.RestoreHandler(snapshot)

	p.logger.WithFields(logrus.Fields{
		"state_hash": stateHash,
		"err":        err,
	}).Debug("InmemProxy.Restore")

	return err
}

// OnStateChanged calls the StateChangeHandler.
func (p *InmemProxy) OnStateChanged(state state.State) error {
	return p.handler.StateChangeHandler(state)
}
