package dummy

import (
	"time"

	"github.com/mosaicnetworks/babble/src/dummy/state"
	socket "github.com/mosaicnetworks/babble/src/proxy/socket/babble"
	"github.com/sirupsen/logrus"
)

//DummySocketClient is a socket implementation of the dummy app. Babble and the
//app run in separate processes and communicate through TCP sockets using
//a SocketBabbleProxy and a SocketAppProxy.
type DummySocketClient struct {
	state       *state.State
	babbleProxy *socket.SocketBabbleProxy
	logger      *logrus.Logger
}

//NewDummySocketClient instantiates a DummySocketClient and starts the
//SocketBabbleProxy
func NewDummySocketClient(clientAddr string, nodeAddr string, logger *logrus.Logger) (*DummySocketClient, error) {

	babbleProxy, err := socket.NewSocketBabbleProxy(nodeAddr, clientAddr, 1*time.Second, logger)
	if err != nil {
		return nil, err
	}

	state := state.NewState(logger)

	client := &DummySocketClient{
		state:       state,
		babbleProxy: babbleProxy,
		logger:      logger,
	}

	go client.Run()

	return client, nil
}

//Run listens for messages from Babble via the SocketProxy
func (c *DummySocketClient) Run() {
	for {
		select {
		case commit := <-c.babbleProxy.CommitCh():
			c.logger.Debug("CommitBlock")
			stateHash, err := c.state.CommitBlock(commit.Block)
			commit.Respond(stateHash, err)
		case snapshotRequest := <-c.babbleProxy.SnapshotRequestCh():
			c.logger.Debug("GetSnapshot")
			snapshot, err := c.state.GetSnapshot(snapshotRequest.BlockIndex)
			snapshotRequest.Respond(snapshot, err)
		case restoreRequest := <-c.babbleProxy.RestoreCh():
			c.logger.Debug("Restore")
			stateHash, err := c.state.Restore(restoreRequest.Snapshot)
			restoreRequest.Respond(stateHash, err)
		}
	}
}

//SubmitTx sends a transaction to Babble via the SocketProxy
func (c *DummySocketClient) SubmitTx(tx []byte) error {
	return c.babbleProxy.SubmitTx(tx)
}
