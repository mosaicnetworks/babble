package app

import (
	bcrypto "github.com/mosaicnetworks/babble/crypto"
	"github.com/mosaicnetworks/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

//InmemProxy is used for testing
type InmemAppProxy struct {
	submitCh              chan []byte
	stateHash             []byte
	committedTransactions [][]byte
	logger                *logrus.Logger
}

func NewInmemAppProxy(logger *logrus.Logger) *InmemAppProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	return &InmemAppProxy{
		submitCh:              make(chan []byte),
		stateHash:             []byte{},
		committedTransactions: [][]byte{},
		logger:                logger,
	}
}

func (iap *InmemAppProxy) commit(block hashgraph.Block) ([]byte, error) {

	iap.committedTransactions = append(iap.committedTransactions, block.Transactions()...)

	hash := iap.stateHash
	for _, t := range block.Transactions() {
		tHash := bcrypto.SHA256(t)
		hash = bcrypto.SimpleHashFromTwoHashes(hash, tHash)
	}

	iap.stateHash = hash

	//XXX - To test fast-forward before snapshot feature
	//return iap.stateHash, nil
	return []byte{}, nil

}

//------------------------------------------------------------------------------
//Implement AppProxy Interface

func (p *InmemAppProxy) SubmitCh() chan []byte {
	return p.submitCh
}

func (p *InmemAppProxy) CommitBlock(block hashgraph.Block) (stateHash []byte, err error) {
	p.logger.WithFields(logrus.Fields{
		"round_received": block.RoundReceived(),
		"txs":            len(block.Transactions()),
	}).Debug("InmemProxy CommitBlock")
	return p.commit(block)
}

//------------------------------------------------------------------------------

func (p *InmemAppProxy) SubmitTx(tx []byte) {
	p.submitCh <- tx
}

func (p *InmemAppProxy) GetCommittedTransactions() [][]byte {
	return p.committedTransactions
}
