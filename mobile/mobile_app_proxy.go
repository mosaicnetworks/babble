package mobile

import (
	"github.com/babbleio/babble/hashgraph"
	"github.com/sirupsen/logrus"
)

// AppProxy object
type AppProxy struct {
	submitCh chan []byte
	logger   *logrus.Logger
	sub      *Subscription
}

// NewAppProxy create proxy
func NewAppProxy(sub *Subscription, logger *logrus.Logger) (proxy *AppProxy) {
	proxy = &AppProxy{
		submitCh: make(chan []byte),
		logger:   logger,
		sub:      sub,
	}
	return
}

// Implement AppProxy Interface

// SubmitCh the node is listening for new transactions on this channel
func (p *AppProxy) SubmitCh() chan []byte {
	return p.submitCh
}

// CommitBlock notify the client for new blocks
func (p *AppProxy) CommitBlock(block hashgraph.Block) error {
	p.sub.commitTx(1, block.Transactions)
	return nil
}
