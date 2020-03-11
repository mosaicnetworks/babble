package wamp

import (
	"strings"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
)

const certFile = "test_data/cert.pem"
const keyFile = "test_data/key.pem"

func TestWamp(t *testing.T) {
	url := "localhost:1443"
	realm := "office"
	certFile := "test_data/cert.pem"
	keyFile := "test_data/key.pem"

	server, err := NewServer(url,
		realm,
		certFile,
		keyFile,
		common.NewTestLogger(t, logrus.DebugLevel).WithField("component", "signal-server"))

	if err != nil {
		t.Fatal(err)
	}

	go server.Run()
	defer server.Shutdown()
	// Allow the server some time to run otherwise we get some connection
	// refused errors
	time.Sleep(time.Second)

	callee, err := NewClient(url,
		realm,
		"callee",
		certFile,
		common.NewTestLogger(t, logrus.DebugLevel).WithField("component", "signal-client callee"))

	if err != nil {
		t.Fatal(err)
	}
	defer callee.Close()

	if err := callee.Listen(); err != nil {
		t.Fatal(err)
	}

	caller, err := NewClient(
		url,
		realm,
		"caller",
		certFile,
		common.NewTestLogger(t, logrus.DebugLevel).WithField("component", "signal-client caller"))

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
