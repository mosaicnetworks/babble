package wamp

import (
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
	callee.Listen()

	caller, err := NewClient(url, "office", "caller")
	if err != nil {
		t.Fatal(err)
	}
	defer caller.Close()

	caller.Offer("callee", webrtc.SessionDescription{})
}
