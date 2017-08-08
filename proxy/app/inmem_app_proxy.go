package app

import "github.com/Sirupsen/logrus"

//InmemProxy is used for testing
type InmemAppProxy struct {
	submitCh    chan []byte
	commitedTxs [][]byte
	logger      *logrus.Logger
}

func NewInmemAppProxy(logger *logrus.Logger) *InmemAppProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	return &InmemAppProxy{
		submitCh:    make(chan []byte),
		commitedTxs: [][]byte{},
		logger:      logger,
	}
}

func (p *InmemAppProxy) SubmitCh() chan []byte {
	return p.submitCh
}

func (p *InmemAppProxy) CommitTx(tx []byte) error {
	p.logger.WithField("tx", tx).Debug("InmemProxy CommitTx")
	p.commitedTxs = append(p.commitedTxs, tx)
	return nil
}

//-------------------------------------------------------
//Implement AppProxy Interface

func (p *InmemAppProxy) SubmitTx(tx []byte) {
	p.submitCh <- tx
}

func (p *InmemAppProxy) GetCommittedTransactions() [][]byte {
	return p.commitedTxs
}
