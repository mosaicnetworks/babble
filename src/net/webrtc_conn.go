package net

import (
	"net"
	"time"

	"github.com/pion/datachannel"
)

// WebRTCConn implements net.Conn around a webrtc datachannel
type WebRTCConn struct {
	dataChannel datachannel.ReadWriteCloser
}

// NewWebRTCConn instantiates a WebRTCConn from a datachannel
func NewWebRTCConn(dataChannel datachannel.ReadWriteCloser) *WebRTCConn {
	return &WebRTCConn{
		dataChannel: dataChannel,
	}
}

// Read implements the Conn Read method.
func (c *WebRTCConn) Read(p []byte) (int, error) {
	return c.dataChannel.Read(p)
}

// Write implements the Conn Write method.
func (c *WebRTCConn) Write(p []byte) (int, error) {
	return c.dataChannel.Write(p)
}

// Close implements the Conn Close method.
func (c *WebRTCConn) Close() error {
	return c.dataChannel.Close()
}

// LocalAddr is a stub
func (c *WebRTCConn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr is a stub
func (c *WebRTCConn) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline is a stub
func (c *WebRTCConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a stub
func (c *WebRTCConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a stub
func (c *WebRTCConn) SetWriteDeadline(t time.Time) error {
	return nil
}
