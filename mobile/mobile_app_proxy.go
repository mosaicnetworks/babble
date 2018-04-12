package mobile

import (
	"github.com/babbleio/babble/hashgraph"
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

// CommitBlock commits a Block's transactions one by one.
// gomobile cannot export a Block object because it doesn't support arrays of
// arrays of bytes
func (p *mobileAppProxy) CommitBlock(block hashgraph.Block) error {
	for _, tx := range block.Transactions {
		p.commitHandler.OnCommit(tx)
	}
	return nil
}
