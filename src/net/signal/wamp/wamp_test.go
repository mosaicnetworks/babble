package wamp

import (
	"strings"
	"testing"

	"github.com/pion/webrtc/v2"
)

func TestWamp(t *testing.T) {
	url := "localhost:8000"

	server, err := NewServer(url, "office")
	if err != nil {
		t.Fatal(err)
	}

	go server.Run()
	defer server.Shutdown()

	callee, err := NewClient(url, "office", "callee")
	if err != nil {
		t.Fatal(err)
	}
	defer callee.Close()

	if err := callee.Listen(); err != nil {
		t.Fatal(err)
	}

	caller, err := NewClient(url, "office", "caller")
	if err != nil {
		t.Fatal(err)
	}
	defer caller.Close()

	// We expect the call to reach the callee and to generate an
	// ErrProcessingOffer error because the SDP is empty. We are only trying to
	// test that the RPC call is relayed and that the handler on the receiving
	// end is called
	_, err = caller.Offer("callee", webrtc.SessionDescription{})
	if err == nil || !strings.Contains(err.Error(), ErrProcessingOffer) {
		t.Fatal("Should have receveived an ErrProcessingOffer")

	}
}
