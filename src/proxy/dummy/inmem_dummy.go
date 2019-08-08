package dummy

import (
	"github.com/mosaicnetworks/babble/src/proxy/inmem"
	"github.com/sirupsen/logrus"
)

// InmemDummyClient is an in-memory implementation of the dummy app. It actually
// imlplements the AppProxy interface, and can be passed in the Babble
// constructor directly
type InmemDummyClient struct {
	*inmem.InmemProxy
	state  *State
	logger *logrus.Entry
}

//NewInmemDummyClient instantiates an InemDummyClient
func NewInmemDummyClient(logger *logrus.Entry) *InmemDummyClient {
	state := NewState(logger)

	proxy := inmem.NewInmemProxy(state, logger)

	client := &InmemDummyClient{
		InmemProxy: proxy,
		state:      state,
		logger:     logger,
	}

	return client
}

//SubmitTx sends a transaction to the Babble node via the InmemProxy
func (c *InmemDummyClient) SubmitTx(tx []byte) {
	c.InmemProxy.SubmitTx(tx)
}

//GetCommittedTransactions returns the state's list of transactions
func (c *InmemDummyClient) GetCommittedTransactions() [][]byte {
	return c.state.GetCommittedTransactions()
}
