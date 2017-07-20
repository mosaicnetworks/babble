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

import "io"

// RPCResponse captures both a response and a potential error.
type RPCResponse struct {
	Response interface{}
	Error    error
}

// RPC has a command, and provides a response mechanism.
type RPC struct {
	Command  interface{}
	Reader   io.Reader
	RespChan chan<- RPCResponse
}

// Respond is used to respond with a response, error or both
func (r *RPC) Respond(resp interface{}, err error) {
	r.RespChan <- RPCResponse{resp, err}
}

// Transport provides an interface for network transports
// to allow a node to communicate with other nodes.
type Transport interface {
	// Consumer returns a channel that can be used to
	// consume and respond to RPC requests.
	Consumer() <-chan RPC

	// LocalAddr is used to return our local address to distinguish from our peers.
	LocalAddr() string

	// Sync sends the appropriate RPC to the target node.
	Sync(target string, args *SyncRequest, resp *SyncResponse) error

	EagerSync(target string, args *EagerSyncRequest, resp *EagerSyncResponse) error

	// Close permanently closes a transport, stopping
	// any associated goroutines and freeing other resources.
	Close() error
}

// WithPeers is an interface that a transport may provide which allows for connection and
// disconnection.
// "Connect" is likely to be nil.
type WithPeers interface {
	Connect(peer string, t Transport) // Connect a peer
	Disconnect(peer string)           // Disconnect a given peer
	DisconnectAll()                   // Disconnect all peers, possibly to reconnect them later
}

// LoopbackTransport is an interface that provides a loopback transport suitable for testing
// e.g. InmemTransport. It's there so we don't have to rewrite tests.
type LoopbackTransport interface {
	Transport // Embedded transport reference
	WithPeers // Embedded peer management
}
