package inapp

import (
	"time"

	"github.com/mosaicnetworks/babble/src/proxy/proto"
	"github.com/sirupsen/logrus"
)

type DummyInappClient struct {
	state  *proto.State
	proxy  *InappProxy
	logger *logrus.Logger
}

func NewDummyInappClient(logger *logrus.Logger) (*DummyInappClient, error) {
	proxy := NewInappProxy(1*time.Second, logger)

	state := proto.NewState(logger)

	state.WriteMessage([]byte("InappDummy"))

	client := &DummyInappClient{
		state:  state,
		proxy:  proxy,
		logger: logger,
	}

	go client.Run()

	return client, nil
}

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

func (c *DummyInappClient) SubmitTx(tx []byte) error {
	return c.proxy.SubmitTx(tx)
}
