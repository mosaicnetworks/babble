package net

import (
	"fmt"
	"net"
	"time"

	"github.com/mosaicnetworks/babble/src/net/signal"
	"github.com/pion/datachannel"
	webrtc "github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
)

// WebRTCStreamLayer implements the StreamLayer interface for WebRTC
type WebRTCStreamLayer struct {
	peerConnections        map[string]*webrtc.PeerConnection
	dataChannels           map[uint16]datachannel.ReadWriteCloser
	signal                 signal.Signal
	incomingConnAggregator chan net.Conn
	logger                 *logrus.Entry
}

// NewWebRTCStreamLayer instantiates a new WebRTCStreamLayer and fires up the
// background connection aggregator (signaling process)
func NewWebRTCStreamLayer(signal signal.Signal, logger *logrus.Entry) *WebRTCStreamLayer {
	stream := &WebRTCStreamLayer{
		peerConnections:        make(map[string]*webrtc.PeerConnection),
		dataChannels:           make(map[uint16]datachannel.ReadWriteCloser),
		signal:                 signal,
		incomingConnAggregator: make(chan net.Conn),
		logger:                 logger,
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

			w.logger.Debug("WebRTCStreamLayer Processing Offer")

			peerConnection, err := w.newPeerConnection(w.incomingConnAggregator, false)
			if err != nil {
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

// newPeerConnection creates a PeerConnection and pipes corresponding
// DataChannel connections into the provided channel. The createDataChannel
// paratemer determines whether a new DataChannel is created for the
// PeerConnection or if we just bind to its OnDataChannel handler. Basically,
// set it to true when actively creating a PeerConnection (you are making the
// offer) and vice-versa.
func (w *WebRTCStreamLayer) newPeerConnection(connCh chan net.Conn, createDataChannel bool) (*webrtc.PeerConnection, error) {
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
		w.logger.WithField("state", connectionState.String()).Debug("ICE Connection State has changed")
	})

	if createDataChannel {
		// Create a datachannel with label 'data'
		dataChannel, err := peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			return nil, err
		}

		w.pipeDataChannel(dataChannel, connCh)
	} else {
		// Register data channel creation handling
		peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			w.pipeDataChannel(d, connCh)
		})
	}

	return peerConnection, nil
}

func (w *WebRTCStreamLayer) pipeDataChannel(dataChannel *webrtc.DataChannel, connCh chan net.Conn) error {
	// Register channel opening handling
	dataChannel.OnOpen(func() {
		// XXX why are these not firing?
		dataChannel.OnClose(func() {
			w.logger.Debug("XXX DataChannel OnClose")
		})

		dataChannel.OnError(func(err error) {
			w.logger.Debugf("XXX DataChannel OnError: %v", err)
		})

		// Detach the data channel
		raw, err := dataChannel.Detach()
		if err != nil {
			w.logger.WithError(err).Error("Error detaching DataChannel")
		}

		// XXX
		w.dataChannels[*dataChannel.ID()] = raw

		connCh <- NewWebRTCConn(raw)
	})

	return nil
}

// Dial implements the StreamLayer interface.
// - Create/Get PeerConnection associated with address (public key?)
// - Create a DataChannel and detatch it in it's OnOpen handler
// - ICE negotiation
// - Return a net.Conn wrapping the detached datachannel
func (w *WebRTCStreamLayer) Dial(target string, timeout time.Duration) (net.Conn, error) {
	// connCh is a channel for receiving net.Conn objects asynchronously when
	// the DataChannel's OnOpen callback is fired.
	connCh := make(chan net.Conn)

	// Create or get PeerConnection and pipe DataChannel connections to connCh
	pc, err := w.newPeerConnection(connCh, true)
	if err != nil {
		return nil, err
	}

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

	// synchronous offer/answer RPC request through signal to exchange SDP
	// information.
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
	case conn := <-connCh:
		return conn, nil
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

	// Close all data channels
	for _, dc := range w.dataChannels {
		dc.Close()
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
