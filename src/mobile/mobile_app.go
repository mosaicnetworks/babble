package mobile

import (
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/proxy"
	"github.com/sirupsen/logrus"
)

/*
This type is not exported
*/

// mobileApp implements the AppProxy interface.
type mobileApp struct {
	commitHandler    CommitHandler
	exceptionHandler ExceptionHandler
	logger           *logrus.Entry
}

func newMobileApp(commitHandler CommitHandler,
	exceptionHandler ExceptionHandler,
	logger *logrus.Entry) *mobileApp {
	mobileApp := &mobileApp{
		commitHandler:    commitHandler,
		exceptionHandler: exceptionHandler,
		logger:           logger,
	}
	return mobileApp
}

// CommitHandler implements the AppProxy interface. It encodes the Blocks with
// JSON to pass them to and from the mobile application.
func (m *mobileApp) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	blockBytes, err := block.Marshal()
	if err != nil {
		m.logger.Debug("mobileAppProxy error marhsalling Block")
		return proxy.CommitResponse{}, err
	}

	processedBlockBytes := m.commitHandler.OnCommit(blockBytes)

	processedBlock := new(hashgraph.Block)
	err = processedBlock.Unmarshal(processedBlockBytes)
	if err != nil {
		m.logger.Debug("mobileAppProxy error unmarshalling processed Block")
		return proxy.CommitResponse{}, err
	}

	response := proxy.CommitResponse{
		StateHash:                   processedBlock.StateHash(),
		InternalTransactionReceipts: processedBlock.InternalTransactionReceipts(),
	}

	return response, nil
}

// SnapshotHandler implements the AppProxy interface.
func (m *mobileApp) SnapshotHandler(blockIndex int) ([]byte, error) {
	return []byte{}, nil
}

// RestoreHandler implements the AppProxy interface.
func (m *mobileApp) RestoreHandler(snapshot []byte) ([]byte, error) {
	return []byte{}, nil
}
