// Package node implements the reactive component of a Babble node.
//
// This is the part of Babble that controls the gossip routines and accesses the
// underlying hashgraph to execute the consensus algorithm. Node implements a
// state machine where the states are defined in the state package.
//
// Gossip
//
// Babble nodes communicate with other Babble nodes in a fully connected p2p
// network. Nodes gossip by repeatedly choosing another node at random and
// telling eachother what they know about the hashgraph. The gossip protocol is
// extremely simple and serves the dual purpose of gossiping about transactions
// and about the gossip itself (the hashgraph). The hashgraph contains enough
// information to compute a consensus ordering of transactions.
//
// The communication mechanism is a custom RPC protocol over network transport
// as defined in the net package. It implements a Pull-Push gossip system which
// relies on two RPC commands: Sync and EagerSync. When node A wants to sync
// with node B, it sends a SyncRequest to B containing a description of what it
// knows about the hashgraph. B computes what it knows that A doesn't know and
// returns a SyncResponse with the corresponding events in topological order.
// Upon receiving the SyncResponse, A updates its hashgraph accordingly and
// calculates the consensus order. Then, A sends an EagerSyncRequest to B with
// the Events that it knows and B doesn't know. Upon receiving the
// EagerSyncRequest, B updates its hashgraph and runs the consensus methods.
//
// Dynamic Membership
//
// A Babble node is started with a genesis peer-set (genesis.peers.json) and a
// current peer-set (peers.json). If its public-key does not belong to the
// current peer-set, the node will enter the Joining state, where it will
// attempt to join the peer-set. It does so by signing an InternalTransaction
// and submitting it for consensus via a JoinRequest to one of the current
// nodes.
//
// If the JoinRequest was successful, and the new node makes it into the
// peer-set, it will either enter the Babbling state directly, or the CatchingUp
// state, depending on whether the FastSync option was enabled. In the former
// case, the node will have to retrieve the entire history of the hashgraph and
// commit all the blocks it missed one-by-one. This is where it is important to
// have the genesis peer-set, to allow the joining node to validate the
// consensus decisions and peer-set changes from the beginning of the hashgraph.
//
// The protocol for removing peers is almost identical, with the difference that
// a node submits a LeaveRequest for itself upon capturing a SIGINT signal when
// the Babble process is terminated cleanly.
//
// FastSync
//
// When a node is started and belongs to the current validator-set, it will
// either enter the Babbling state, or the CatchingUp state, depending on
// whether the FastSync option was enablied in the configuration.
//
// In the CatchingUp state, a node determines the best node to fast-sync from
// (the node which has the longest hashgraph) and attempts to fast-forward to
// their last consensus snapshot, until the operation succeeds. Hence, FastSync
// introduces a new type of command in the communication protocol: FastForward.
//
// Upon receiving a FastForwardRequest, a node must respond with the last
// consensus snapshot, as well as the corresponding Hashgraph section (the
// Frame) and Block. With this information, and having verified the Block
// signatures against the other items as well as the known validator set, the
// requesting node attempts to reset its Hashgraph from the Frame, and restore
// the application from the snapshot.
package node
