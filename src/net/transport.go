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

	// Sync, EagerSync, FastForward, and Join send the appropriate RPC to the
	//target node.

	Sync(target string, args *SyncRequest, resp *SyncResponse) error

	EagerSync(target string, args *EagerSyncRequest, resp *EagerSyncResponse) error

	FastForward(target string, args *FastForwardRequest, resp *FastForwardResponse) error

	Join(target string, args *JoinRequest, resp *JoinResponse) error

	// Close permanently closes a transport, stopping
	// any associated goroutines and freeing other resources.
	Close() error
}
