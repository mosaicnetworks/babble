package dummy

import (
	"time"

	"github.com/mosaicnetworks/babble/src/dummy/state"
	"github.com/mosaicnetworks/babble/src/proxy/inapp"
	"github.com/sirupsen/logrus"
)

// DummyInappClient is an in-memory implmentation of the dummy app. It uses an
// InappProxy to communicate with a Babble node.
type DummyInappClient struct {
	state  *state.State
	proxy  *inapp.InappProxy
	logger *logrus.Logger
}

//NewDummyInappClient instantiates a DummyInappClient
func NewDummyInappClient(logger *logrus.Logger) (*DummyInappClient, error) {
	proxy := inapp.NewInappProxy(1*time.Second, logger)

	state := state.NewState(logger)

	client := &DummyInappClient{
		state:  state,
		proxy:  proxy,
		logger: logger,
	}

	go client.Run()

	return client, nil
}

//Run listens for messages from the Babble node via the InappProxy
func (c *DummyInappClient) Run() {
	for {
		select {
		case commit := <-c.proxy.CommitCh():
			c.logger.Debug("CommitBlock")
			stateHash, err := c.state.CommitBlock(commit.Block)
			commit.Respond(stateHash, err)
		case snapshotRequest := <-c.proxy.SnapshotRequestCh():
			c.logger.Debug("GetSnapshot")
			snapshot, err := c.state.GetSnapshot(snapshotRequest.BlockIndex)
			snapshotRequest.Respond(snapshot, err)
		case restoreRequest := <-c.proxy.RestoreCh():
			c.logger.Debug("Restore")
			stateHash, err := c.state.Restore(restoreRequest.Snapshot)
			restoreRequest.Respond(stateHash, err)
		}
	}
}

//SubmitTx sends a transaction to the Babble node via the InappProxy
func (c *DummyInappClient) SubmitTx(tx []byte) error {
	return c.proxy.SubmitTx(tx)
}
