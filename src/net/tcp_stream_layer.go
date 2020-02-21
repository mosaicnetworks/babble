package net

import (
	"net"
	"time"
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
