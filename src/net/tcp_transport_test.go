package net

import (
	"testing"

	"github.com/mosaicnetworks/babble/src/common"
)

func TestTCPTransport_BadAddr(t *testing.T) {
	_, err := NewTCPTransport("0.0.0.0:0", "", 1, 0, 0, common.NewTestEntry(t, common.TestLogLevel))
	if err != errNotAdvertisable {
		t.Fatalf("err: %v", err)
	}
}

func TestTCPTransport_WithAdvertise(t *testing.T) {
	trans, err := NewTCPTransport("0.0.0.0:0", "127.0.0.1:12345", 1, 0, 0, common.NewTestEntry(t, common.TestLogLevel))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if trans.LocalAddr() != "127.0.0.1:12345" {
		t.Fatalf("bad: %v", trans.LocalAddr())
	}
}
