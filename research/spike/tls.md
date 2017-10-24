
# TLS spike 2017-09-26

Can we handle inter-node encryption and authentication at the transport layer
by using e.g. TLS instead of implementing MACs and possibly encryption at the
application layer?

## Questions

- How can we fit our P2P model on top of TLS' client-server model?
- Are client certificates useful/necessary for this?
- What does the go TLS API look like?
- Does it support dynamically updating trusted certificates without restarting
  the process? Somewhat, need to define a mechanism for propagation.

## Answers

The `Config` type is responsible for setting up certificates. After it's passed
to one of the TLS functions it should not be modified. I assume this means we
would have to close a connection and reconnect with an updated configuration in
the event of an update.

However, instead of static certificate lists it looks like you can set up
`GetCertificate` and `GetClientCertificate` hook functions instead.

There is also the matter of how these updates are propagated on top of the
application-level protocol.

Moreover there is an additional verification mechanism for certificates, called
after normal verification (or if normal verification is disabled).

I guess one problem with TLS certificates is that they are issued for specific
host names?

ECDHE handshakes are also supported. To ensure interoperability with the node
keys, we would have to restrict ourselves to the curves supported by TLS 1.2 or
1.3 (working draft as of July 2017). Unfortunately it looks like only ECDSA is
supported for signatures, no EdDSA, not sure whether this is a problem though.

## Links

- https://golang.org/pkg/crypto/tls/#Config.BuildNameToCertificate
- https://gist.github.com/denji/12b3a568f092ab951456
- https://en.wikipedia.org/wiki/Transport_Layer_Security#TLS_1.3_.28draft.29

