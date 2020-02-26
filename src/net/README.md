# Net

This package contains various implementations of the Transport interface, which 
is used by a Babble node to communicate with other nodes. The transport is used
to send and receive RPC requests (SyncRequest, EagerSyncRequest, JoinRequest,
etc.).

There are three implementations:

- Inmem: in-memory transport used only for testing
- TCP: communicating over plain TCP
- WebRTC: using WebRTC

Because Babble is a peer-to-peer application, it can run into issues with NATs
and firewalls. 

The TCP transport is suitable when nodes are in the same local network, or when
users are able to configure their connections appropriately (ex port-forwarding)
which is not always possible (ex cellular networks).

The WebRTC transport is supposed to address the NAT traversal issue and to 
address the problem of connecting peers regardless of how many routers are 
between them. 

WebRTC requires a signaling mechanism for peers to exchange connection 
information. Usually, this would be implemented in a centralized server. So when
the WebRTC transport is used, Babble is not fully p2p anymore. That being said,
all the  computation and data at the application layer remains p2p; the 
signaling server is only used as a sort of peer-discovery mechanism.