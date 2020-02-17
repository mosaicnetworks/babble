package net

import (
	"context"
	"net"
	"time"

	"github.com/pion/ice"
)

// WebRTCStreamLayer implements the StreamLayer interface for WebRTC
type WebRTCStreamLayer struct {
	agent *ice.Agent
}

// Dial implements the StreamLayer interface
func (t *WebRTCStreamLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	return t.agent.Dial(context.TODO(), "", "")
}

// Accept implements the net.Listener interface
func (t *WebRTCStreamLayer) Accept() (c net.Conn, err error) {
	return t.agent.Accept(context.TODO(), "", "")
}

// Close implements the net.Listener interface
func (t *WebRTCStreamLayer) Close() (err error) {
	return t.agent.Close()
}

// Addr implements the net.Listener interface
func (t *WebRTCStreamLayer) Addr() net.Addr {
	return nil
}

// AdvertiseAddr implements the net.Listener interface
func (t *WebRTCStreamLayer) AdvertiseAddr() net.Addr {
	return nil
}
