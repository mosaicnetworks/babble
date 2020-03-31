package net

import (
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
)

func TestWebRTCStreamLayerWithWampSignal(t *testing.T) {
	url := nextWampAddress()

	server, err := wamp.NewServer(url, realm, certFile, keyFile, common.NewTestEntry(t, common.TestLogLevel))
	if err != nil {
		t.Fatal(err)
	}

	go server.Run()
	defer server.Shutdown()
	time.Sleep(time.Second)

	wampSignal1, err := wamp.NewClient(url, realm, "alice", certFile, false, common.NewTestEntry(t, common.TestLogLevel))
	if err != nil {
		t.Fatal(err)
	}

	wampSignal2, err := wamp.NewClient(url, realm, "bob", certFile, false, common.NewTestEntry(t, common.TestLogLevel))
	if err != nil {
		t.Fatal(err)
	}

	stream1 := NewWebRTCStreamLayer(wampSignal1, common.NewTestEntry(t, common.TestLogLevel))
	defer stream1.Close()

	go func() {
		err := stream1.listen()
		if err != nil {
			t.Fatal(err)
		}
	}()

	stream2 := NewWebRTCStreamLayer(wampSignal2, common.NewTestEntry(t, common.TestLogLevel))
	defer stream2.Close()

	_, err = stream2.Dial("alice", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
}
