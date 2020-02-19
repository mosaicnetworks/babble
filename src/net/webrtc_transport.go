package net

import (
	"time"

	"github.com/sirupsen/logrus"
)

// NewWebRTCTransport returns a NetworkTransport that is built on top of
// a WebRTC streaming transport layer, with log output going to the supplied
// Logger
func NewWebRTCTransport(
	addr string,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	logger *logrus.Entry,
) (*NetworkTransport, error) {
	return newWebRTCTransport(addr, maxPool, timeout, joinTimeout, func(stream StreamLayer) *NetworkTransport {
		return NewNetworkTransport(stream, maxPool, timeout, joinTimeout, logger)
	})
}

func newWebRTCTransport(
	addr string,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	transportCreator func(stream StreamLayer) *NetworkTransport) (*NetworkTransport, error) {

	signal := NewTestSignal(addr)

	// Create stream
	stream := NewWebRTCStreamLayer(signal)

	go stream.listen()

	// Create the network transport
	trans := transportCreator(stream)
	return trans, nil
}
