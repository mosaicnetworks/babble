package net

import (
	"os"
	"testing"
	"time"
)

func TestWebRTCStreamLayer(t *testing.T) {
	os.RemoveAll("test_data")
	os.Mkdir("test_data", os.ModeDir|0777)

	testSignal1 := NewTestSignal("test_data/offer.sdp", "test_data/answer.sdp")
	testSignal2 := NewTestSignal("test_data/offer.sdp", "test_data/answer.sdp")

	stream1 := NewWebRTCStreamLayer(testSignal1)
	go func() {
		err := stream1.listen()
		if err != nil {
			t.Fatal(err)
		}
	}()

	stream2 := NewWebRTCStreamLayer(testSignal2)

	_, err := stream2.Dial("test", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

}
