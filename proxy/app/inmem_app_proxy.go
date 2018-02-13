package app

import (
	"github.com/babbleio/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

//InmemProxy is used for testing
type InmemAppProxy struct {
	submitCh             chan []byte
	commitedTransactions [][]byte
	logger               *logrus.Logger
}

func NewInmemAppProxy(logger *logrus.Logger) *InmemAppProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	return &InmemAppProxy{
		submitCh:             make(chan []byte),
		commitedTransactions: [][]byte{},
		logger:               logger,
	}
}

//------------------------------------------------------------------------------
//Implement AppProxy Interface

func (p *InmemAppProxy) SubmitCh() chan []byte {
	return p.submitCh
}

func (p *InmemAppProxy) CommitBlock(block hashgraph.Block) error {
	p.logger.WithFields(logrus.Fields{
		"round_received": block.RoundReceived(),
		"txs":            len(block.Transactions()),
	}).Debug("InmemProxy CommitBlock")
	p.commitedTransactions = append(p.commitedTransactions, block.Transactions()...)
	return nil
}

//------------------------------------------------------------------------------

func (p *InmemAppProxy) SubmitTx(tx []byte) {
	p.submitCh <- tx
}

func (p *InmemAppProxy) GetCommittedTransactions() [][]byte {
	return p.commitedTransactions
}
