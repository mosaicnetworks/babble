.. _blockchain:

From Hashgraph to Blockchain
============================

This document describes a technique for projecting a hashgraph onto a
blockchain, which is better suited for representing an immutable ordered list
of transactions. In this system, the order is governed by the Hashgraph
consensus algorithm but the transactions are mapped onto a linear data
structure composed of blocks; each block containing an ordered list of
transactions, a hash of the resulting application state, a hash of the
corresponding section of the hashgraph (Frame), a hash of the current peer-set,
and a collection of signatures from the set of validators. This method enables
hashgraph-based systems to implement any Inter-Blockchain Communication
protocol and integrate with an Internet of Blockchains.

Motivation
----------

The consumable output of any consensus system is an ordered list of
transactions. Developers have been using blockchains to model such lists
because they are efficient to work with. A linear data structure composed of
batches of transactions, hashed and signed together, enabling easy verification
of any transaction, is the right tool for the job. Although the word blockchain
is now used in a much broader sense, it originally designated a data structure.
Consensus algorithms, public/private networks, cryptocurrencies, etc., are
independent concepts.

Hashgraph is a beautiful consensus algorithm based on a homonymous data
structure. The hashgraph data structure, however, is not easy to work with when
it comes to representing a linear sequence of transactions. It is a Directed
Acyclic Graph (DAG) from which the order must be extracted via some complex
consensus functions. To verify the consensus index of a given transaction, one
has to re-compute the consensus methods on a subset of the hashgraph. On the
other hand, blockchains do not need any further processing to extract the
ordered list of transactions and simple cryptographic primitives are sufficient
to validate blocks.

The "hashgraph vs blockchain" debate is a red herring. Blockchain is just a
data structure; the engine is the underlying consensus algorithm. The
projection method exposes an easy-to-work-with blockchain powered by the
efficient Hashgraph consensus algorithm.

Implementation
--------------

::

    |   |   |  w53
    |   |   | / |
    |   |  w52  |
    |   | / |   |                                     ^
    |  w51  |   |                                     |
    | / |   |   |                                     | 
   w50  |   |   |-----------------------  ---------------------------
    |   \   |   |                         - Block 5 | (Block 4 Hash)-
    |   |   \   |                         ---------------------------
    |   |   |  w43                        - E: [w40, w41, w42, w43] -
    |   |   | / |                         ---------------------------
    |   |  w42  |                         - S: [S50, S51, S52, S53] -
    |   | / |   |                         ---------------------------
    |  w41  |   |                                     |
    | / |   |   |                                     |
   w40  |   |   |-----------------------  ---------------------------
    |   \   |   |                         - Block 4 | (Block 3 Hash)-
    |   |   \   |                         ---------------------------
    |   |   |  w33                        - E: [w30, w31, w32, w33] -
    |   |   | / |                         ---------------------------
    |   |  w32  |                         - S: [S40, S41, S42, S43] -
    |   | / |   |                         ---------------------------
    |  w31  |   |                                     |
    | / |   |   |                                     |
   w30  |   |   |-----------------------  --------------------------- 
    |   \   |   |                         - Block 3 | (Block 2 Hash)-
    |   |   \   |                         ---------------------------
    |   |   |  w23                        - E: [w20, w21, w22, w23] -
    |   |   | / |                         ---------------------------
    |   |  w22  |                         - S: [S30, S31, S32, S33] -
    |   | / |   |                         ---------------------------
    |  w21  |   |                                     |
    | / |   |   |                                     |
   w20  |   |   |-----------------------  ---------------------------  
    |   \   |   |                         - Block 2 | (Block 1 Hash)-  
    |   |   \   |                         ---------------------------
    |   |   |  w13                        - E: [w10, w11, w12, w13] -
    |   |   | / |                         ---------------------------
    |   |  w12  |                         - S: [S20, S21, S22, S23] -
    |   | / |   |                         ---------------------------
    |  w11  |   |                                     |
    | / |   |   |                                     |
   w10  |   |   |-----------------------  ---------------------------
    |   \   |   |                         - Block 1                 -
    |   |   \   |                         ---------------------------
    |   |   |  e32                        - E: [w00, w01, w02, w03, -
    |   |   | / |                         -     e10, e21, e32]      -
    |   |  e21  |                         ---------------------------
    |   | / |   |                         - S: [S10, S11, S12, S13] - 
    |  e10  |   |                         ---------------------------
    | / |   |   |
   w00 w01 w02 w03
    0   1   2   3

    Caption:
    --------

    E: List of Events contained in Block. Here, we mention Events because it is
    easier to represent than transactions. Blocks would actually contain only
    the transactions of the Events, but that is complicated to represent in this
    diagram.

    Sij: Signature of Block i by validator j

The Hashgraph algorithm always commits Events in batches. Indeed, when the fame
of a super-majority of witnesses from a given round is decided, all the Events
that are seen by all these famous witnesses (but not from an earlier round) get
assigned the same *Round Received* and sorted according to a deterministic
function. At that point, the consensus order of these Events is decided and
will not change.

We gather the transactions of all the Events from the same *Round Received*
into blocks. When Events get assigned a *Round Received* and sorted, we package
their transactions (in canonical order) into a block and commit that block to
the application. The application returns a hash of the state obtained by
applying the block's transactions sequentially and we append this hash to the
block's body before signing it. Block signatures will be exchanged as part of
the regular gossip routine and appended to their corresponding blocks as they
are received from other peers if they match the local block. Once a block has
collected signatures from at least 1/3 of validators, it is deemed accepted
because, by hypothesis, at least one of those signatures originates from an
honest peer.

We extend the Event data structure to contain a set of block-signatures by the
Event's creator. Having assigned a *RoundReceived* to a set of Events and
produced a corresponding block, a member will append the block's signature in
the next Event it defines. Hence, block-signatures piggy-back on the regular
gossip messages and propagate at the same speed. Upon receiving Events from an
other peer, a member will verify their block-signatures against its own version
of the blocks; if the signatures match, they are recorded with the block. With
this extended gossip routine, nodes simultaneously build up the hashgraph and
the corresponding blockchain. It preserves the simplicity of the hashgraph
system, which is one of its most valuable features, by not adding new types of
messages; it only extends the existing Event data-structure.

By construction, the fame of a round R witness can only be decided by a witness
in round R+2 or above. Hence, when a block is created for a *Round Received* R
(block R), the hashgraph already contains Events at round R+2 or more; the
signatures for block R, will be gossiped at the same time as Events of round
R+2 or more. It follows that the signatures of block R will arrive with a lag
of 2 or more consensus rounds.

Block Structure
---------------

.. code:: go

  Block: {
      Body:{
          Index                       int                          // block index
	      RoundReceived               int                          // round received of corresponding hashgraph frame
	      Timestamp                   int64                        // unix timestamp (median of timestamps in round-received famous witnesses)
	      StateHash                   []byte                       // root hash of the application after applying block payload; to be populated by application Commit
	      FrameHash                   []byte                       // hash of corresponding hashgraph frame
	      PeersHash                   []byte                       // hash of peer-set
	      Transactions                [][]byte                     // transaction payload
	      InternalTransactions        []InternalTransaction        // internal transaction payload (add/remove peers)
	      InternalTransactionReceipts []InternalTransactionReceipt // receipts for internal transactions; to be populated by application Commit
      }
      Signatures: map[string]string
  }
 
Blocks contain a body and a set of signatures. Signatures are based on the hash
of the body; which is enough to verify the entire block because it contains a
digital fingerprint of the body.

The Body's *RoundReceived* corresponds to the *RoundReceived* of the hashgraph 
Events who's transactions are included in the block; it serves the purpose of 
tying back to the underlying hashgraph. We do not produce a block when all the 
Events of a *Round Received* are empty. Hence, two consecutive blocks may have 
non-consecutive RoundReceived values and we use an additional property to index 
the blocks.

The 'Timestamp' is a Unix timestamp (number of seconds since January 1st, 1970)
corresponding to the median of the timestamps of the famous witnesses in the 
frame's round-received. Upon creating a hashgraph Event, a Unix timestamp is 
automatically added to it using the creator's system clock. The Block's 
timestamp is the median of the timestamps included in the famous witnesses of
the block's round-received. Note that nodes may have non-synchronised clocks, 
and may purposefuly tinker with their clocks to bias the block timestamp.

The FrameHash corresponds to the Frame in the hashgraph at RoundReceived. It is
used in the FastSync protocol to verify the relationship between the Block and
the Frame returned in a FastForwardResponse.

The body also contains a hash of the application's state resulting from
applying the block's transactions sequentially. Thus, with the consenus
algorithm and the necessary assumption that at least two thirds of participants
are not compromised, collecting signatures from at least one third of
validators provides sufficient evidence that all honest nodes have applied the
same transactions in the same order, and computed the same state.

With the new Dynamic Membership protocol, which enables adding and removing
peers dynamically, we added a PeersHash field to the body, to keep track of the
validator-set. We can check the Frame's peer-set against the block's PeersHash
to ensure that we are counting signatures from the appropriate peer-set.

InternalTransactions and InternalTransactionReceipts are used to track attempts
to update the peer-set. InternalTransactions encode requests to join or leave
the peer-set. Upon receiving a CommitBlock message, the application can accept
or refuse InternalTransactions by returning correponding
InternalTransactionReceipts.

Enhancements
------------

Inter-Blockchain Communication
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Inter-Blockchain Communication (IBC) is about verifying on one chain that a
transaction happened on another chain; one blockchain acts as a light-client to
another blockchain. It is much simpler to build a light-client for a blockchain
than for a hashgraph. In an effort to enable interoperability between
blockchains, several initiatives have been proposed to build protocols for IBC
like Cosmos, Polkadot and EOS. The projection method allows hashgraph-based
systems to integrate with these network architectures.
