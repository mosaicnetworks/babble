// Package signal implements different signalling mechanisms for WebRTC. A
// signalling mechanism is a way for peers to exchange connection metadata
// prior to establishing a direct p2p connection.
package signal

import "github.com/pion/webrtc/v2"

// Signal defines an interface for systems to send and receive SDP offers and
// answers.
type Signal interface {
	// ID returns the identifier of this end of a connection
	ID() string

	// Listen is an ansynchronous function that sets up listeners to listen for
	// incoming SDP offers and forward them to to the Consumer channel.
	Listen() error

	// Consumer is the channel through which incoming SDP offers are forwarded
	// to the WebRTCStreamLayer. SDP offers are wrapped around a promise object
	// which offers a response mechanism.
	Consumer() <-chan OfferPromise

	// Offer sends an SDP offer and waits for an SDP answer
	Offer(target string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error)

	// Close closes the signal
	Close() error
}
