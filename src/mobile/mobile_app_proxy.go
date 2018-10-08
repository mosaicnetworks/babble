package mobile

import (
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy/inapp"
	"github.com/sirupsen/logrus"
)

/*
This type is not exported
*/

// mobileAppProxy object
type mobileAppProxy struct {
	*inapp.InappProxy

	commitHandler    CommitHandler
	exceptionHandler ExceptionHandler
	logger           *logrus.Logger
}

// newMobileAppProxy create proxy
func newMobileAppProxy(
	commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	logger *logrus.Logger,
) *mobileAppProxy {
	return &mobileAppProxy{
		InappProxy:       inapp.NewInappProxy(time.Second, logger),
		commitHandler:    commitHandler,
		exceptionHandler: exceptionHandler,
		logger:           logger,
	}
}

// CommitBlock commits a Block's to the App and expects the resulting state hash
// gomobile cannot export a Block object because it doesn't support arrays of
// arrays of bytes; so we have to serialize the block.
// Overrides  InappProxy::CommitBlock
func (p *mobileAppProxy) CommitBlock(block hashgraph.Block) ([]byte, error) {
	blockBytes, err := block.Marshal()

	if err != nil {
		p.logger.Debug("mobileAppProxy error marhsalling Block")

		return nil, err
	}

	stateHash := p.commitHandler.OnCommit(blockBytes)

	return stateHash, nil
}
