package mobile

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy/inmem"
	"github.com/sirupsen/logrus"
)

/*
This type is not exported
*/

// mobileAppProxy object
type mobileAppProxy struct {
	*inmem.InmemProxy

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

	// gomobile cannot export a Block object because it doesn't support arrays of
	// arrays of bytes; so we have to serialize the block.
	commitHandlerFunc := func(block hashgraph.Block) ([]byte, error) {
		blockBytes, err := block.Marshal()
		if err != nil {
			logger.Debug("mobileAppProxy error marhsalling Block")
			return nil, err
		}
		stateHash := commitHandler.OnCommit(blockBytes)
		return stateHash, nil
	}

	snapshotHandlerFunc := func(blockIndex int) ([]byte, error) {
		return []byte{}, nil
	}

	restoreHandlerFunc := func(snapshot []byte) ([]byte, error) {
		return []byte{}, nil
	}

	return &mobileAppProxy{
		InmemProxy: inmem.NewInmemProxy(commitHandlerFunc,
			snapshotHandlerFunc,
			restoreHandlerFunc,
			logger),
		commitHandler:    commitHandler,
		exceptionHandler: exceptionHandler,
		logger:           logger,
	}
}
