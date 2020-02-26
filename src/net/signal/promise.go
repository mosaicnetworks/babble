package signal

import (
	"github.com/pion/webrtc/v2"
)

// OfferPromise provides a mechanism to asynchronously process and respond to a
// WebRTC SDP Offer. It contains the SDP offer, the sender's identifier, and a
// response channel.
type OfferPromise struct {
	From     string
	Offer    webrtc.SessionDescription
	RespChan chan<- OfferPromiseResponse
}

// OfferPromiseResponse is the object returned through an OfferPromise. It wraps
// an SDP answer and a potential error.
type OfferPromiseResponse struct {
	Answer *webrtc.SessionDescription
	Error  error
}

// Respond is used to respond to a OfferPromise.
func (p *OfferPromise) Respond(answer *webrtc.SessionDescription, err error) {
	p.RespChan <- OfferPromiseResponse{answer, err}
}
