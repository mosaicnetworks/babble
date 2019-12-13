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

// TCPStreamLayer implements StreamLayer interface for plain TCP.
type TCPStreamLayer struct {
	advertise string
	listener  *net.TCPListener
}

// Dial implements the StreamLayer interface.
func (t *TCPStreamLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, timeout)
}

// Accept implements the net.Listener interface.
func (t *TCPStreamLayer) Accept() (c net.Conn, err error) {
	return t.listener.Accept()
}

// Close implements the net.Listener interface.
func (t *TCPStreamLayer) Close() (err error) {
	lnFile, _ := t.listener.File()

	if err := t.listener.Close(); err != nil {
		return err
	}

	if lnFile != nil {
		if err := lnFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Addr implements the net.Listener interface.
func (t *TCPStreamLayer) Addr() net.Addr {
	return t.listener.Addr()
}

// AdvertiseAddr implements the SteamLayer interface.
func (t *TCPStreamLayer) AdvertiseAddr() string {
	// Use an advertise addr if provided
	if t.advertise != "" {
		return t.advertise
	}
	return t.listener.Addr().String()
}

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
