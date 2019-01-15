package mobile

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

/*
This type is not exported
*/

//mobileApp implements the ProxyHandler interface
type mobileApp struct {
	commitHandler    CommitHandler
	exceptionHandler ExceptionHandler
	logger           *logrus.Logger
}

func newMobileApp(commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	logger *logrus.Logger) *mobileApp {
	mobileApp := &mobileApp{
		commitHandler:    commitHandler,
		exceptionHandler: exceptionHandler,
		logger:           logger,
	}
	return mobileApp
}

func (m *mobileApp) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	blockBytes, err := block.Marshal()
	if err != nil {
		m.logger.Debug("mobileAppProxy error marhsalling Block")
		return proxy.CommitResponse{}, err
	}

	stateHash := m.commitHandler.OnCommit(blockBytes)

	commitResponse := proxy.CommitResponse{
		StateHash: stateHash,
	}

	return commitResponse, nil
}

func (m *mobileApp) SnapshotHandler(blockIndex int) ([]byte, error) {
	return []byte{}, nil
}

func (m *mobileApp) RestoreHandler(snapshot []byte) ([]byte, error) {
	return []byte{}, nil
}
