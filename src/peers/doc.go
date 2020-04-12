// Package peers defines the concept of a Babble peer and implements functions
// to manage collections of peers.
//
// A Babble peer is an entity that operates a Babble node. When a peer belongs
// to a collection of peers engaged in a Babble network, we refer to it as a
// validator, or a participant, to indicate that it is actively involved in the
// underlying consensus algorithm. The distinction is relevant because a peer
// can be in or out of consensus network as the group membership evolves.
//
// A peer-set refers to a collection of peers, whilst a validator-set refers to
// a collection of peers simultaneously involved in the same consensus network.
// Again, the distinction is relevant because a collection of peers forming a
// validator-set at a given time in a Babble network, may become out of date as
// peers join or leave the network.
//
// Peers are identified by their public keys, and optionaly a moniker which is a
// non-unique user-friendly name. When WebRTC is not activated, a peer should
// also specify an IP address and port where it can be reached by other peers.
// With WebRTC, the public key is enough to indentify a peer within the
// signaling server, and the network address is not necessary.
//
// Upon starting up, Babble expects to find a peers.json file, and optionaly a
// peers.genesis.json file, in its data directory. The peers.json file
// represents a list of peers that the node should attempt to connect to. The
// peers.genesis.json file defines the initial validator-set corresponding to
// the inception of the Babble network. If a  peers.genesis.json file is not
// found, the peers.json file is used for the initial validator-set. The
// reason for having two peers files is explained in the node package,
package peers
