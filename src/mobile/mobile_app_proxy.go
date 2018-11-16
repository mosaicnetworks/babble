package mobile

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy"
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

func (m *mobileAppProxy) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	blockBytes, err := block.Marshal()

	if err != nil {
		m.logger.Debug("mobileAppProxy error marhsalling Block")

		return proxy.CommitResponse{}, err
	}

	commitResponse := m.commitHandler.OnCommit(blockBytes)

	return commitResponse, nil
}

func (m *mobileAppProxy) SnapshotHandler(blockIndex int) ([]byte, error) {
	return []byte{}, nil
}

func (m *mobileAppProxy) RestoreHandler(snapshot []byte) ([]byte, error) {
	return []byte{}, nil
}

// newMobileAppProxy create proxy
func newMobileAppProxy(
	commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	logger *logrus.Logger,
) *mobileAppProxy {
	mobileApp := &mobileAppProxy{
		commitHandler:    commitHandler,
		exceptionHandler: exceptionHandler,
		logger:           logger,
	}

	mobileApp.InmemProxy = inmem.NewInmemProxy(mobileApp, logger)

	return mobileApp
}
