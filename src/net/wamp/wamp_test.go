package wamp

import (
	"testing"
)

func TestWamp(t *testing.T) {
	url := "localhost:8000"

	server, err := NewServer(url, "office")
	if err != nil {
		t.Fatal(err)
	}

	go server.Run()

	callee, err := NewClient(url, "office", "callee")
	if err != nil {
		t.Fatal(err)
	}
	defer callee.Close()

	caller, err := NewClient(url, "office", "caller")
	if err != nil {
		t.Fatal(err)
	}
	defer caller.Close()

	caller.Call("callee", "am", "stram", "gram")

	server.Shutdown()
}
