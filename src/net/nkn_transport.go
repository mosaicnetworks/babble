package net

import (
	"fmt"
	"time"

	"github.com/nknorg/nkn-sdk-go"
	"github.com/sirupsen/logrus"
)

// NewNKNTransport implements a NetworkTransport that is built on top of a NKN
// StreamLayer.
func NewNKNTransport(
	nknAccount *nkn.Account,
	nknBaseIdentifier string,
	nknNumSubClients int,
	nknConfig *nkn.ClientConfig,
	nknConnectTimeout time.Duration,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	logger *logrus.Entry,
) (*NetworkTransport, error) {
	return newNKNTransport(
		nknAccount,
		nknBaseIdentifier,
		nknNumSubClients,
		nknConfig,
		nknConnectTimeout,
		func(stream StreamLayer) *NetworkTransport {
			return NewNetworkTransport(stream, maxPool, timeout, joinTimeout, logger)
		},
	)
}

func newNKNTransport(
	account *nkn.Account,
	baseIdentifier string,
	numSubClients int,
	config *nkn.ClientConfig,
	connectTimeout time.Duration,
	transportCreator func(stream StreamLayer) *NetworkTransport,
) (*NetworkTransport, error) {

	multiclient, err := nkn.NewMultiClient(account, baseIdentifier, numSubClients, false, config)
	if err != nil {
		return nil, err
	}

	select {
	case <-time.After(connectTimeout):
		return nil, fmt.Errorf("timeout waiting to connect to nkn")
	case <-multiclient.OnConnect.C:
		break
	}

	err = multiclient.Listen(nil)
	if err != nil {
		return nil, err
	}

	stream := &nknStreamLayer{
		multiclient: multiclient,
	}

	return transportCreator(stream), nil
}
