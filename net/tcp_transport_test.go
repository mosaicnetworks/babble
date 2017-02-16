/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package net

import (
	"net"
	"testing"

	"github.com/arrivets/babble/common"
)

func TestTCPTransport_BadAddr(t *testing.T) {
	_, err := NewTCPTransportWithLogger("0.0.0.0:0", nil, 1, 0, common.NewTestLogger(t))
	if err != errNotAdvertisable {
		t.Fatalf("err: %v", err)
	}
}

func TestTCPTransport_WithAdvertise(t *testing.T) {
	addr := &net.TCPAddr{IP: []byte{127, 0, 0, 1}, Port: 12345}
	trans, err := NewTCPTransportWithLogger("0.0.0.0:0", addr, 1, 0, common.NewTestLogger(t))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if trans.LocalAddr() != "127.0.0.1:12345" {
		t.Fatalf("bad: %v", trans.LocalAddr())
	}
}
