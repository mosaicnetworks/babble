package signal

import (
	"github.com/pion/webrtc/v2"
)

// OfferPromiseResponse is the object returned throug an OfferPromise. It wraps
// an SDP answer and a potential error.
type OfferPromiseResponse struct {
	Answer *webrtc.SessionDescription
	Error  error
}

// OfferPromise provides a mechanism to asynchronously process and respond to a
// WebRTC SDP Offer. It contains the SDP offer, the identifier of the originator
// of the offer, and a response channel.
type OfferPromise struct {
	From     string
	Offer    webrtc.SessionDescription
	RespChan chan<- OfferPromiseResponse
}

// Respond is used to respond with an SDP answer, and/or an error.
func (p *OfferPromise) Respond(answer *webrtc.SessionDescription, err error) {
	p.RespChan <- OfferPromiseResponse{answer, err}
}
