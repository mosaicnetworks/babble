package net

import (
	"net"
	"time"
)

// tcpStreamLayer implements StreamLayer interface for plain TCP.
type tcpStreamLayer struct {
	advertise string
	listener  *net.TCPListener
}

// Accept implements the net.Listener interface.
func (t *tcpStreamLayer) Accept() (c net.Conn, err error) {
	return t.listener.Accept()
}

// Close implements the net.Listener interface.
func (t *tcpStreamLayer) Close() (err error) {
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
func (t *tcpStreamLayer) Addr() net.Addr {
	return t.listener.Addr()
}

// Dial implements the StreamLayer interface.
func (t *tcpStreamLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, timeout)
}

// AdvertiseAddr implements the SteamLayer interface.
func (t *tcpStreamLayer) AdvertiseAddr() string {
	// Use an advertise addr if provided
	if t.advertise != "" {
		return t.advertise
	}
	return t.listener.Addr().String()
}
