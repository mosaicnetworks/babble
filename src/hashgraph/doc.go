// Package hashgraph implements the distributed consensus algorithm.
//
// Babble is based on the Hashgraph consensus algorithm, invented by Leemon
// Baird:
//
// http://www.swirlds.com/downloads/SWIRLDS-TR-2016-01.pdf
//
// http://www.swirlds.com/downloads/SWIRLDS-TR-2016-02.pdf
//
// Store
//
// The hashgraph is a data structure that contains all the information about the
// history of the gossip and thereby continuously grows as gossip spreads. There
// are various strategies to keep the size of the hashgraph limited. In our
// implementation, the Hashgraph object has a dependency on a Store object which
// contains the actual data and is abstracted behind an interface.
//
// There are currently two implementations of the Store interface. InmemStore
// uses a set of in-memory LRU caches which can be extended to persist stale
// items to disk and the size of the LRU caches is configurable. BadgerStore is
// a wrapper around this cache that also persists objects to a key-value store
// on disk. The database produced by the BadgerStore can be reused to bootstrap
// a node back to a specific state.
//
// Blocks
//
// Babble projects the hashgraph DAG onto a linear data structure composed of
// blocks. The consensus algorithm operates on the DAG but the consumable output
// is an ordered list of blocks. Each block contains an ordered list of
// transactions, a hash of the resulting application state, a hash of the
// corresponding section of the hashgraph (Frame), a hash of the current
// peer-set, and a collection of signatures from the set of validators. This
// method enables hashgraph-based systems to implement light-weight clients
// to audit and verify blocks without implementing the full consensus algorithm.
package hashgraph
