package net

import (
	"fmt"
	"net"
	"time"

	"github.com/mosaicnetworks/babble/src/net/signal"
	"github.com/pion/datachannel"
	webrtc "github.com/pion/webrtc/v2"
)

// WebRTCStreamLayer implements the StreamLayer interface for WebRTC
type WebRTCStreamLayer struct {
	peerConnections        map[string]*webrtc.PeerConnection
	signal                 signal.Signal
	incomingConnAggregator chan net.Conn
}

// NewWebRTCStreamLayer instantiates a new WebRTCStreamLayer and fires up the
// background connection aggregator (signaling process)
func NewWebRTCStreamLayer(signal signal.Signal) *WebRTCStreamLayer {
	stream := &WebRTCStreamLayer{
		peerConnections:        make(map[string]*webrtc.PeerConnection),
		signal:                 signal,
		incomingConnAggregator: make(chan net.Conn),
	}
	return stream
}

// Receive SDP offers from Signal, create corresponding PeerConnections and
// respond. The PeerConnection's DataChannel is piped into the connection
// aggregator.
func (w *WebRTCStreamLayer) listen() error {
	// Start the Signal listener
	go w.signal.Listen()

	consumer := w.signal.Consumer()

	// Process incoming offers
	for {
		select {
		case offerPromise := <-consumer:

			fmt.Println("WebRTCStreamLayer Processing Offer")

			peerConnection, err := newPeerConnection()
			if err != nil {
				return err
			}

			// Forward it's DataChannel to the connection aggregator
			if err := w.pipePeerConnection(peerConnection); err != nil {
				return err
			}

			// Set the remote SessionDescription
			err = peerConnection.SetRemoteDescription(offerPromise.Offer)
			if err != nil {
				return err
			}

			// Create answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				return err
			}

			// Sets the LocalDescription, and starts our UDP listeners
			err = peerConnection.SetLocalDescription(answer)
			if err != nil {
				return err
			}

			offerPromise.Respond(&answer, nil)

			w.peerConnections[offerPromise.From] = peerConnection
		}
	}
}

// newPeerConnection creates a PeerConnection
func newPeerConnection() (*webrtc.PeerConnection, error) {
	// Create a SettingEngine and enable Detach
	s := webrtc.SettingEngine{}
	s.DetachDataChannels()

	// Create an API object with the engine
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s))

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection using the API object
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	return peerConnection, nil
}

// pipePeerConnection forwards a PeerConnection's opened DataChannels into the
// incoming connection aggregator
func (w *WebRTCStreamLayer) pipePeerConnection(pc *webrtc.PeerConnection) error {
	// Register data channel creation handling
	pc.OnDataChannel(func(d *webrtc.DataChannel) {

		// Register channel opening handling
		d.OnOpen(func() {
			// Detach the data channel
			raw, dErr := d.Detach()
			if dErr != nil {
				panic(dErr)
			}

			w.incomingConnAggregator <- NewWebRTCConn(raw)
		})
	})

	return nil
}

// Dial implements the StreamLayer interface.
// - Create/Get PeerConnection associated with address (public key?)
// - Create a DataChannel and detatch it in it's OnOpen handler
// - ICE negotiation
// - Return a net.Conn wrapping the detached datachannel
func (w *WebRTCStreamLayer) Dial(target string, timeout time.Duration) (net.Conn, error) {

	if _, ok := w.peerConnections[target]; ok {
		return nil, fmt.Errorf("Already dialed")
	}

	// Create or get PeerConnection
	pc, err := newPeerConnection()
	if err != nil {
		return nil, err
	}

	// Create a datachannel with label 'data'
	dataChannel, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	resCh := make(chan datachannel.ReadWriteCloser)
	// Register channel opening handling
	dataChannel.OnOpen(func() {
		// Detach the data channel
		raw, dErr := dataChannel.Detach()
		if dErr != nil {
			panic(dErr)
		}

		resCh <- raw
	})

	// Create an offer to send to the signaling system
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = pc.SetLocalDescription(offer)
	if err != nil {
		return nil, err
	}

	answer, err := w.signal.Offer(target, offer)
	if err != nil {
		return nil, err
	}

	if answer == nil {
		return nil, fmt.Errorf("No answer")
	}

	// Apply the answer as the remote description
	err = pc.SetRemoteDescription(*answer)
	if err != nil {
		return nil, err
	}

	w.peerConnections[target] = pc

	// Wait for DataChannel opening
	timer := time.After(timeout)
	select {
	case <-timer:
		return nil, fmt.Errorf("Dial timeout")
	case raw := <-resCh:
		return NewWebRTCConn(raw), nil
	}
}

// Accept consumes the incoming connection aggregator fed by the 'listen'
// routine. It aggregates the connections from all DataChannels formed with
// PeerConnections.
func (w *WebRTCStreamLayer) Accept() (c net.Conn, err error) {
	nextConn := <-w.incomingConnAggregator
	return nextConn, nil
}

// Close implements the net.Listener interface. It closes the Signal and all the
// PeerConnections
func (w *WebRTCStreamLayer) Close() (err error) {
	// Close the connection to the signal server
	w.signal.Close()

	// Close all peer connections
	for _, pc := range w.peerConnections {
		pc.Close()
	}
	return nil
}

// Addr implements the net.Listener interface
func (w *WebRTCStreamLayer) Addr() net.Addr {
	return nil
}

// AdvertiseAddr implements the net.Listener interface
func (w *WebRTCStreamLayer) AdvertiseAddr() string {
	return w.signal.ID()
}
