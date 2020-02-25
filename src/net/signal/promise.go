package signal

import (
	"github.com/pion/webrtc/v2"
)

// OfferPromiseResponse is a struct returned by an OfferPromise
type OfferPromiseResponse struct {
	Answer *webrtc.SessionDescription
	Error  error
}

// OfferPromise provides a mechanism to asynchronously process and respond to a
// WebRTC Offer
type OfferPromise struct {
	From     string
	Offer    webrtc.SessionDescription
	RespChan chan<- OfferPromiseResponse
}

// Respond is used to respond with a response, error or both
func (p *OfferPromise) Respond(answer *webrtc.SessionDescription, err error) {
	p.RespChan <- OfferPromiseResponse{answer, err}
}
