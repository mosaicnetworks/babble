// Package net implements different transports to communicate between Babble
// nodes.
//
// This package contains various implementations of the Transport interface,
// which is used by Babble nodes to send and receive RPC requests (SyncRequest,
// EagerSyncRequest, JoinRequest, etc.). There are three implementations:
//
// - Inmem: in-memory transport used only for testing
//
// - TCP: communicating over plain TCP
//
// - WebRTC: using WebRTC
//
// TCP
//
// The TCP transport is suitable when nodes are in the same local network, or
// when users are able to configure their connections appropriately to avoid NAT
// issues.
//
// To use a TCP transport, set the following configuration options in the Babble
// Config object (cf config package):
//
// - BinAdddr: the IP:PORT of the TCP socket that Babble binds to.
//
// - AdvertiseAddr: (optional) The address that is advertised to other nodes. If
// BindAddr is a local address not reachable by other peers, it is usefull to
// set AdvertiseAddr to the reachable public address.
//
// WebRTC
//
// Because Babble is a peer-to-peer application, it can run into issues with
// NATs and firewalls. The WebRTC transport addresses the NAT traversal issue,
// but it requires centralised servers for peers to exchange connection
// information and to provide STUN/TURN services.
//
// To use a WebRTC transport, use the following properties in the Config object:
//
// - WebRTC: tells Babble to use a WebRTC transport
//
// - SignalAddr: address of the WebRTC signaling server
//
// - SignalRealm: routing domain within the signaling server
//
// WebRTC requires a signaling mechanism for peers to exchange connection
// information. Usually, this would be implemented in a centralized server. So
// when the WebRTC transport is used, Babble is not fully p2p anymore. That
// being said, all the  computation and data at the application layer remains
// p2p; the signaling server is only used as a sort of peer-discovery mechanism.
//
// The WebRTCTransport ignores the BindAddr and AdvertiseAddr configuration
// values, but it requires the address of the signaling server. This address is
// specified by the SignalAddr configuration value. The SignalRealm defines a
// domain within the signaling server, such that signaling messages are only
// routed withing this domain.
package net
