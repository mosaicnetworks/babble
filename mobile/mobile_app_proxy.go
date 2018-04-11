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
	submitCh     chan []byte
	logger       *logrus.Logger
	subscription *Subscription
}

// newMobileAppProxy create proxy
func newMobileAppProxy(subscription *Subscription,
	logger *logrus.Logger) (proxy *mobileAppProxy) {
	proxy = &mobileAppProxy{
		submitCh:     make(chan []byte),
		logger:       logger,
		subscription: subscription,
	}
	return
}

// Implement AppProxy Interface

// SubmitCh is the channel through which the App sends transactions to the node.
func (p *mobileAppProxy) SubmitCh() chan []byte {
	return p.submitCh
}

// CommitBlock commits a new Block to the App
func (p *mobileAppProxy) CommitBlock(block hashgraph.Block) error {
	p.subscription.OnCommit(block)
	return nil
}
