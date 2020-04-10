package net

// Transport provides an interface for network transports
// to allow a node to communicate with other nodes.
type Transport interface {

	// Starts the transport listening
	Listen()

	// Consumer returns a channel that can be used to
	// consume and respond to RPC requests.
	Consumer() <-chan RPC

	// LocalAddr is used to return our local address
	LocalAddr() string

	// AdvertiseAddr is used to return our advertise address where other peers
	// can reach us
	AdvertiseAddr() string

	// Sync, EagerSync, FastForward, and Join send the appropriate RPC to the
	// target node.

	Sync(target string, args *SyncRequest, resp *SyncResponse) error

	EagerSync(target string, args *EagerSyncRequest, resp *EagerSyncResponse) error

	FastForward(target string, args *FastForwardRequest, resp *FastForwardResponse) error

	Join(target string, args *JoinRequest, resp *JoinResponse) error

	// Close permanently closes a transport, stopping
	// any associated goroutines and freeing other resources.
	Close() error
}
