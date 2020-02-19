package net

import (
	"os"
	"testing"
	"time"
)

func TestWebRTCStreamLayer(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	testSignal1 := NewTestSignal("alice")
	testSignal2 := NewTestSignal("bob")

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
