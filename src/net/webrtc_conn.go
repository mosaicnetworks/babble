package net

import (
	"io"
	"net"
	"sync/atomic"
	"time"
)

// WebRTCConn implement net.Conn
type WebRTCConn struct {
	bytesReceived uint64
	bytesSent     uint64
	reader        io.Reader
	writer        io.Writer
}

// Read implements the Conn Read method.
func (c *WebRTCConn) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	atomic.AddUint64(&c.bytesReceived, uint64(n))
	return n, err
}

// Write implements the Conn Write method.
func (c *WebRTCConn) Write(p []byte) (int, error) {
	atomic.AddUint64(&c.bytesSent, uint64(len(p)))
	return c.writer.Write(p)
}

// Close implements the Conn Close method. It is used to close the connection.
func (c *WebRTCConn) Close() error {
	return nil
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
