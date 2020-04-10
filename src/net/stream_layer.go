package net

import (
	"net"
	"time"
)

// StreamLayer is used with the NetworkTransport to provide the low level stream
// abstraction.
type StreamLayer interface {
	net.Listener

	// Dial is used to create a new outgoing connection
	Dial(address string, timeout time.Duration) (net.Conn, error)

	// AdvertiseAddr returns the publicly-reachable address of the stream
	AdvertiseAddr() string
}
