package net

import (
	"net"
	"time"

	"github.com/pion/datachannel"
)

// webRTCConn implements net.Conn around a webrtc datachannel.
type webRTCConn struct {
	dataChannel datachannel.ReadWriteCloser
}

// newWebRTCConn instantiates a webRTCConn from a datachannel.
func newWebRTCConn(dataChannel datachannel.ReadWriteCloser) *webRTCConn {
	return &webRTCConn{
		dataChannel: dataChannel,
	}
}

// Read implements the Conn Read method.
func (c *webRTCConn) Read(p []byte) (int, error) {
	return c.dataChannel.Read(p)
}

// Write implements the Conn Write method.
func (c *webRTCConn) Write(p []byte) (int, error) {
	return c.dataChannel.Write(p)
}

// Close implements the Conn Close method.
func (c *webRTCConn) Close() error {
	return c.dataChannel.Close()
}

// LocalAddr is a stub
func (c *webRTCConn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr is a stub
func (c *webRTCConn) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline is a stub
func (c *webRTCConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a stub
func (c *webRTCConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a stub
func (c *webRTCConn) SetWriteDeadline(t time.Time) error {
	return nil
}
