// Package wamp implements a WebRTC signaling system using RPC over WebSockets.
//
// This package  contains a WAMP server that relays RPC requests between
// connected clients, and a client which implements the Signal interface, and
// which can be used to instantiate a webRTCStreamLayer.
//
// If WebRTC is turned on in the configuration, and Babble finds a cert.pem file
// in its data directory, then Babble will pass this certificate to the signal
// client. Otherwise, it relies on the "web of trust" to validate the server's
// certificate. This means that the certificate can be self-signed because it
// can be passed directly to Babble. There is also an option to skip certificate
// verification, but this should only be used for testing.
package wamp

const (
	// ErrProcessingOffer indicates that the client who received the offer ran
	// into an error while processing it.
	ErrProcessingOffer = "io.babble.processing_offer"
)
