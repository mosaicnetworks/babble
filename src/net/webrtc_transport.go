package net

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc"
)

// WebRTCStreamLayer implements the StreamLayer interface for WebRTC
type WebRTCStreamLayer struct {
	peerConnection webrtc.PeerConnection
}

// Dial implements the StreamLayer interface.
// - Create/Get PeerConnection associated with address (public key?)
// - Create a DataChannel and detatch it in it's OnOpen handler
// - ICE negotiation
// - Return a net.Conn wrapping the detached datachannel
func (w *WebRTCStreamLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	// Create a datachannel with label 'data'
	dataChannel, err := w.peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	resCh := make(chan datachannel.ReadWriteCloser)
	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open.\n", dataChannel.Label(), dataChannel.ID())

		// Detach the data channel
		raw, dErr := dataChannel.Detach()
		if dErr != nil {
			panic(dErr)
		}

		resCh <- raw
	})

	raw := <-resCh

	return NewWebRTCConn(raw), nil
}

// Accept aggregate all the DataChannels from all the PeerConnections
// Be notified when a peer wants to create a peer connection (offer, answer)
// Handle the OnDataChannel, detatch the channel when it's opened and return
// the corresponding Conn
func (w *WebRTCStreamLayer) Accept() (c net.Conn, err error) {
	resCh := make(chan datachannel.ReadWriteCloser)

	// Register data channel creation handling
	w.peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())

			// Detach the data channel
			raw, dErr := d.Detach()
			if dErr != nil {
				panic(dErr)
			}

			resCh <- raw
		})
	})

	raw := <-resCh

	return NewWebRTCConn(raw), nil
}

// Close implements the net.Listener interface
func (w *WebRTCStreamLayer) Close() (err error) {
	return nil
}

// Addr implements the net.Listener interface
func (w *WebRTCStreamLayer) Addr() net.Addr {
	return nil
}

// AdvertiseAddr implements the net.Listener interface
func (w *WebRTCStreamLayer) AdvertiseAddr() net.Addr {
	return nil
}
