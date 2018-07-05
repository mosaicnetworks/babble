package mobile

import (
	"github.com/mosaicnetworks/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

/*
This type is not exported
*/

// mobileAppProxy object
type mobileAppProxy struct {
	submitCh         chan []byte
	logger           *logrus.Logger
	commitHandler    CommitHandler
	exceptionHandler ExceptionHandler
}

// newMobileAppProxy create proxy
func newMobileAppProxy(commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	logger *logrus.Logger) (proxy *mobileAppProxy) {
	proxy = &mobileAppProxy{
		submitCh:         make(chan []byte),
		logger:           logger,
		commitHandler:    commitHandler,
		exceptionHandler: exceptionHandler,
	}
	return
}

// Implement AppProxy Interface

// SubmitCh is the channel through which the App sends transactions to the node.
func (p *mobileAppProxy) SubmitCh() chan []byte {
	return p.submitCh
}

// CommitBlock commits a Block's to the App and expects the resulting state hash
// gomobile cannot export a Block object because it doesn't support arrays of
// arrays of bytes; so we have to serialize the block.
func (p *mobileAppProxy) CommitBlock(block hashgraph.Block) ([]byte, error) {
	blockBytes, err := block.Marshal()
	if err != nil {
		p.logger.Debug("mobileAppProxy error marhsalling Block")
		return nil, err
	}
	stateHash := p.commitHandler.OnCommit(blockBytes)
	return stateHash, nil
}

//TODO - Implement these two functions
func (p *mobileAppProxy) GetSnapshot(blockIndex int) ([]byte, error) {
	return []byte{}, nil
}

func (p *mobileAppProxy) Restore(snapshot []byte) error {
	return nil
}
