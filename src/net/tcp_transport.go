package net

import (
	"errors"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	errNotAdvertisable = errors.New("local bind address is not advertisable")
	errNotTCP          = errors.New("local address is not a TCP address")
)

// NewTCPTransport returns a NetworkTransport that is built on top of
// a TCP streaming transport layer, with log output going to the supplied Logger
func NewTCPTransport(
	bindAddr string,
	advertise string,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	logger *logrus.Entry,
) (*NetworkTransport, error) {
	return newTCPTransport(bindAddr, advertise, maxPool, timeout, joinTimeout, func(stream StreamLayer) *NetworkTransport {
		return NewNetworkTransport(stream, maxPool, timeout, joinTimeout, logger)
	})
}

func newTCPTransport(bindAddr string,
	advertiseAddr string,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	transportCreator func(stream StreamLayer) *NetworkTransport) (*NetworkTransport, error) {

	// Try to bind
	list, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}

	// Try to resolve the advertise address
	var resolvedAdvertise net.Addr
	if advertiseAddr != "" {
		resolvedAdvertise, err = net.ResolveTCPAddr("tcp", advertiseAddr)
		if err != nil {
			return nil, err
		}
	}

	if resolvedAdvertise == nil {
		resolvedAdvertise = list.Addr()
	}

	// Verify that we have a usable advertise address
	addr, ok := resolvedAdvertise.(*net.TCPAddr)
	if !ok {
		list.Close()
		return nil, errNotTCP
	}
	if addr.IP.IsUnspecified() {
		list.Close()
		return nil, errNotAdvertisable
	}

	// Create stream
	stream := &TCPStreamLayer{
		advertise: advertiseAddr,
		listener:  list.(*net.TCPListener),
	}

	// Create the network transport
	trans := transportCreator(stream)
	return trans, nil
}
