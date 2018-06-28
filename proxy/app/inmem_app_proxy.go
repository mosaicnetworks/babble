package app

import (
	"fmt"

	bcrypto "github.com/mosaicnetworks/babble/crypto"
	"github.com/mosaicnetworks/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

//InmemProxy is used for testing
type InmemAppProxy struct {
	submitCh              chan []byte
	stateHash             []byte
	committedTransactions [][]byte
	snapshots             map[int][]byte
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
		snapshots:             make(map[int][]byte),
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

	//XXX do something smart here
	iap.snapshots[block.Index()] = hash

	return iap.stateHash, nil
}

func (iap *InmemAppProxy) restore(snapshot []byte) error {
	//XXX do something smart her
	iap.stateHash = snapshot
	return nil
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

func (p *InmemAppProxy) GetSnapshot(blockIndex int) (snapshot []byte, err error) {
	p.logger.WithField("block", blockIndex).Debug("InmemProxy GetSnapshot")

	snapshot, ok := p.snapshots[blockIndex]
	if !ok {
		return nil, fmt.Errorf("Snapshot %d not found", blockIndex)
	}

	return snapshot, nil
}

func (p *InmemAppProxy) Restore(snapshot []byte) error {
	p.logger.WithField("snapshot", snapshot).Debug("Restore")
	return p.restore(snapshot)
}

//------------------------------------------------------------------------------

func (p *InmemAppProxy) SubmitTx(tx []byte) {
	p.submitCh <- tx
}

func (p *InmemAppProxy) GetCommittedTransactions() [][]byte {
	return p.committedTransactions
}
