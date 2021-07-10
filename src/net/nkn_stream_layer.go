package net

import (
	"net"
	"time"

	"github.com/nknorg/nkn-sdk-go"
)

// nknStreamLayer implements the StreamLayer interface over the nkn network.
type nknStreamLayer struct {
	multiclient *nkn.MultiClient
}

// Accept implements the net.Listener interface.
func (n *nknStreamLayer) Accept() (c net.Conn, err error) {
	return n.multiclient.Accept()
}

// 	 implements the net.Listener interface.
func (n *nknStreamLayer) Close() (err error) {
	return n.multiclient.Close()
}

// Addr implements the net.Listener interface.
func (n *nknStreamLayer) Addr() net.Addr {
	return n.multiclient.Addr()
}

// Dial implements the StreamLayer interface.
func (n *nknStreamLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	dialConfig := &nkn.DialConfig{
		DialTimeout: int32(timeout.Milliseconds()),
	}

	session, err := n.multiclient.DialWithConfig(address, dialConfig)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// AdvertiseAddr implements the StreamLayer interface.
func (n *nknStreamLayer) AdvertiseAddr() string {
	return n.multiclient.Address()
}
