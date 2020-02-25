package signal

import "github.com/pion/webrtc/v2"

// Signal defines an interface for systems to exchange SDP offers and answers
// to establish WebRTC PeerConnections
type Signal interface {
	// Addr returns the local address used to identify this end of a connection
	Addr() string

	// Listen is called to listen for incoming SDP offers, and forward them to
	// to the Consumer channel
	Listen() error

	// Consumer is the channel through which incoming SDP offers are passed to
	// the WebRTCStreamLayer. SDP offers are wrapped around a promise object
	// which offers a response mechanism.
	Consumer() <-chan OfferPromise

	// Offer sends an SDP offer and waits for an answer
	Offer(target string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error)
}
