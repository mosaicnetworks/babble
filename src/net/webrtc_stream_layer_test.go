package net

import (
	"os"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/net/signal/file"
	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
)

func TestWebRTCStreamLayerWithFileSignal(t *testing.T) {

	dir := "test_data/stream"

	os.RemoveAll(dir)
	os.Mkdir(dir, os.ModeDir|0777)

	testSignal1 := file.NewTestSignal("alice", dir)
	testSignal2 := file.NewTestSignal("bob", dir)

	stream1 := NewWebRTCStreamLayer(testSignal1)
	defer stream1.Close()

	go func() {
		err := stream1.listen()
		if err != nil {
			t.Fatal(err)
		}
	}()

	stream2 := NewWebRTCStreamLayer(testSignal2)
	defer stream2.Close()

	_, err := stream2.Dial("alice", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

}

func TestWebRTCStreamLayerWithWampSignal(t *testing.T) {
	url := "localhost:8000"

	server, err := wamp.NewServer(url, "office")
	if err != nil {
		t.Fatal(err)
	}

	go server.Run()
	defer server.Shutdown()

	wampSignal1, _ := wamp.NewClient(url, "office", "alice")
	wampSignal2, _ := wamp.NewClient(url, "office", "bob")

	stream1 := NewWebRTCStreamLayer(wampSignal1)
	defer stream1.Close()

	go func() {
		err := stream1.listen()
		if err != nil {
			t.Fatal(err)
		}
	}()

	stream2 := NewWebRTCStreamLayer(wampSignal2)
	defer stream2.Close()

	_, err = stream2.Dial("alice", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
}
