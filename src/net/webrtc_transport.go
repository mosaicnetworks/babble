package net

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mosaicnetworks/babble/src/net/signal"
)

// NewWebRTCTransport returns a NetworkTransport that is built on top of a
// WebRTC StreamLayer. The signal is a mechanism for peers to exchange
// connection information prior to establishing a direct p2p link.
func NewWebRTCTransport(
	signal signal.Signal,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	logger *logrus.Entry,
) (*NetworkTransport, error) {
	return newWebRTCTransport(signal, maxPool, timeout, joinTimeout, logger, func(stream StreamLayer) *NetworkTransport {
		return NewNetworkTransport(stream, maxPool, timeout, joinTimeout, logger)
	})
}

func newWebRTCTransport(
	signal signal.Signal,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	logger *logrus.Entry,
	transportCreator func(stream StreamLayer) *NetworkTransport) (*NetworkTransport, error) {

	// Create stream
	stream := newWebRTCStreamLayer(signal, logger)

	go stream.listen()

	// Create the network transport
	trans := transportCreator(stream)
	return trans, nil
}
