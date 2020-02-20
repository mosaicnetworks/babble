package net

import (
	"os"
	"testing"
	"time"
)

func TestWebRTCStreamLayer(t *testing.T) {

	dir := "test_data/stream"

	os.RemoveAll(dir)
	os.Mkdir(dir, os.ModeDir|0777)

	testSignal1 := NewTestSignal("alice", dir)
	testSignal2 := NewTestSignal("bob", dir)

	stream1 := NewWebRTCStreamLayer(testSignal1)
	go func() {
		err := stream1.listen()
		if err != nil {
			t.Fatal(err)
		}
	}()

	stream2 := NewWebRTCStreamLayer(testSignal2)

	_, err := stream2.Dial("alice", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

}
