package dummy

import (
	"time"

	socket "github.com/mosaicnetworks/babble/src/proxy/socket/babble"
	"github.com/sirupsen/logrus"
)

// DummySocketClient is a socket implementation of the dummy app. Babble and the
// app run in separate processes and communicate through TCP sockets using
// a SocketBabbleProxy and a SocketAppProxy.
type DummySocketClient struct {
	state       *State
	babbleProxy *socket.SocketBabbleProxy
	logger      *logrus.Entry
}

//NewDummySocketClient instantiates a DummySocketClient and starts the
//SocketBabbleProxy
func NewDummySocketClient(clientAddr string, nodeAddr string, logger *logrus.Entry) (*DummySocketClient, error) {
	state := NewState(logger)

	babbleProxy, err := socket.NewSocketBabbleProxy(nodeAddr, clientAddr, state, 1*time.Second, logger)

	if err != nil {
		return nil, err
	}

	client := &DummySocketClient{
		state:       state,
		babbleProxy: babbleProxy,
		logger:      logger,
	}

	return client, nil
}

//SubmitTx sends a transaction to Babble via the SocketProxy
func (c *DummySocketClient) SubmitTx(tx []byte) error {
	return c.babbleProxy.SubmitTx(tx)
}
