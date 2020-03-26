# Net

This package contains various implementations of the Transport interface, which 
is used by a Babble node to communicate with other nodes. The transport is used
to send and receive RPC requests (SyncRequest, EagerSyncRequest, JoinRequest,
etc.).

There are three implementations:

- Inmem: in-memory transport used only for testing
- TCP: communicating over plain TCP
- WebRTC: using WebRTC

## TCP

The TCP transport is suitable when nodes are in the same local network, or when
users are able to configure their connections appropriately (ex port-forwarding)
which is not always possible (ex cellular networks).

To use a TCP transport, set the following configuration options (through CLI 
flags or babble.toml):

- `--listen`: the IP:PORT of the TCP socket that Babble binds to
- `--advertise`: (optional) The address that is advertised to other nodes. If
               `listen` is a local address not reachable by other peers, it is
                usefull to set `advertise` to the reachable public address.

## WebRTC

Because Babble is a peer-to-peer application, it can run into issues with NATs
and firewalls. The WebRTC transport is supposed to address the NAT traversal 
issue and to address the problem of connecting peers regardless of how many 
routers are between them. 

To use a WebRTC transport, use the following configuration:

- `--webrtc`: tells Babble to use a WebRTC transport
- `--signal-addr`: address of the WebRTC signaling server
- `--signal-realm`: routing domain within the signaling server 

WebRTC requires a signaling mechanism for peers to exchange connection 
information. Usually, this would be implemented in a centralized server. So when
the WebRTC transport is used, Babble is not fully p2p anymore. That being said,
all the  computation and data at the application layer remains p2p; the 
signaling server is only used as a sort of peer-discovery mechanism.

The `WebRTCTransport` ignores the `listen` and `advertise` configuration values,
but it requires the address of the signaling server. This address is specified 
by the `signal-addr` configuration value. The `signal-realm` option defines a 
domain within the signaling server, such that signaling messages are only routed
withing this domain.

The `Signal/` folder contains the code for a Signal server and corresponding 
client. It uses the WAMP protocol, basically RPC over websockets, to relay 
connection information across peers before establishing WebRTC connections. 

The Signal server needs to be started outside of Babble, and should be reachable
at the address specified by `signal-addr`. The `/cmd/signal` folder, from the 
root of this project, contains a standalone executable for the signal server, 
and the `docker/signal` folder contains the corresponding docker image file.

The signal server uses secure websockets, and relies on a TLS certificate. The
server constructor requires both the certificate file and the corresponding key
file.

If `webrtc` is turned on and Babble finds a `cert.pem` file in its `datadir`, 
then it will pass this certificate to the signal client. Otherwise, it relies on
the "web of trust" to validate the server's certificate. This means that the
certificate can be `self-signed` because it can be passed directly to Babble.

