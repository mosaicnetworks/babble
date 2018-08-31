.. _fastsync:

FastSync
========

FastSync is an element of Babble which enables nodes to catch up with other 
nodes without downloading and processing the entire history of gossip (Hashgraph 
+ Blockchain). It is important in the context of mobile ad hoc networks where 
users dynamically create or join groups, and where limited computing resources 
call for periodic pruning of the underlying data store. The solution requires 
nodes to regularly capture snapshots of the application state, and to link each 
snapshot to the section of the Hashgraph it resulted from. A node that 
fell back too far may fast-forward straight to the latest snapshot rather than 
downloading and processing all the transactions it missed. Of course, the 
protocol maintains the BFT properties of the base algorithm by packaging 
relevant data in signed blocks; here again we see the benefits of using a 
blockchain mapping on top of Hashgraph. Although implementing the 
Snapshot/Restore functionality puts extra strain on the application developer, 
it remains entirely optional; FastSync can be activated or deactivated via 
configuration.    

Overview
--------

.. image:: assets/fastsync.png

The Babble node is implemented as a state machine where the possible states are: 
**Babbling**, **CatchingUp**, and **Shutdown**. A node is normally in the 
**Babbling** state where it performs the regular Hashgraph gossip routines, but 
a **sync_limit** response from a peer will trigger the node to enter the 
**CatchingUp** state, where it will attempt to fast-forward to a recent 
snapshot. A **sync_limit** response indicates that the number of Events that the
node needs to download exceeds the **sync_limit** configuration value. 

In the **CatchingUp** state, a node repeatedly chooses another node at random 
(although the above diagram uses the same peer that returned the **sync_limit** 
response) and attempts to fast-forward to their last consensus snapshot, until 
the operation succeeds. Hence, FastSync introduces a new type of command in the 
communication protocol: *FastForward*.

Upon receiving a FastForwardRequest, a node must respond with the last consensus 
snapshot, the section of the Hashgraph it corresponded to (the Frame), and the
coinciding Block. With this information, and having verified the Block 
signatures against the other items as well as the known validator set, the 
requesting node attempts to reset its Hashgraph from the Frame, and restore the 
application from the snapshot. The difficulty resides in defining what is meant 
by *last consensus* snapshot, and how to package enough information in the 
Frames as to form a base for a new/pruned Hashgraph. 

Frames
------

Frames are self-contained sections of the Hashgraph. They are composed of Roots 
and regular Hashgraph Events, where Roots are the base on top of which Events 
can be inserted. Basically, given a Frame, we can initialize a new Hashgraph and 
continue gossiping on top of it; earlier records of the gossip history are 
ignored. 

::

  type Frame struct {
  	Round  int     //RoundReceived
  	Roots  []Root  //[participant ID] => Root
  	Events []Event //Events with RoundReceived = Round
  }

A Frame corresponds to a Hashgraph consensus round. Indeed, the consensus 
algorithm commits Events in batches, which we map onto Frames, and finally onto 
a Blockchain. This is an evolution of the previously defined :ref:`blockchain 
mapping <blockchain>`. Block headers now contain a Frame hash. As we will see 
later, this is useful for security. The Events in a Frame are the Events of the 
corresponding batch, in consensus order.

.. image:: assets/dag_frames_bx.png

Roots
-----

Frames also contain Roots. To get an understanding for why this is necessary, we
must consider the initial state of a Hashgraph, i.e., the base on top of which 
the first Events are inserted. 

The Hashgraph is an interlinked chain of Events, where each Event contains two 
references to anterior Events (SelfParent and OtherParent). Upon inserting an 
Event in the Hashgraph, we check that its references point to existing Events 
(Events that are already in the Hashgraph) and that at least the SelfParent 
reference is not empty. This is partially illustrated in the following picture 
where Event A2 cannot be inserted because its references are unknown. 

.. image:: assets/roots_0.png

So what about the first Event? Until now, we simply implemented a special case, 
whereby the first Event for any participant, could be inserted without checking 
its references. In fact the above picture shows that Events A0, B0, and C0, have
empty references, and yet they are part of the Hashgraph. This special case is 
fine as long as we do not expect to initialize Hashgraphs from a 'non-zero' 
state.

We introduced the concept of Roots to remove the special case and handle more
general situations. They make it possible to initialize a Hashgraph from a 
section of an existing Hashgraph.

.. image:: assets/roots_1.png

A Root is a data structure containing condensed information about the ancestors 
of the first Events to be added to the Hashgraph. Each participant has a Root,
containing a *SelfParent* - the direct ancestor of the first Event for the 
corresponding participant - and *Others* - a map of Event hashes to 
OtherParents. These parents are instances of the **RootEvent** object, which is 
a minimal version of the Hashgraph Event, containing only the information we 
need. RootEvents also contain information about the Index, Round, and 
LamportTimestamp of the corresponding Events. The Root itself contains a 
NextRound field, which helps in calculating the Round of its direct descendant.

::

  type Root struct {
    NextRound  int
    SelfParent RootEvent
    Others     map[string]RootEvent
  }

  type RootEvent struct {
    Hash             string
    CreatorID        int
    Index            int
    LamportTimestamp int
    Round            int
  }

The new rule prescribes that an Event should only be inserted if its parents 
belong to the Hashgraph or are referenced in one of the Roots. The algorithm for 
computing an Event's Round has also changed slightly; there are 6 different 
scenarios to take into consideration when computing the Round of an Event. Each
scenario corresponds to a different relashionship between the Event and its 
creator's Root.

.. image:: assets/round_algo.png

[Explain]

The computation of LamportTimestamp is even easier because it only relies on 
direct parents.

Transition _ could still fail if there are undetermined events below the Frame.
why? Not all Frames can be used to Reset/Fastforward a hashgraph

FastForward
-----------

Given a Frame, we can initialize or reset a Hashgraph to a clean state with 
indexes, rounds, blocks, etc., corresponding to a capture of a live run, such 
that further Events may be inserted and processed independently of past Events. 
This is loosely analogous to IFrames in video encoding, where one can 
fast-forward to any point in the video by downloading a reference IFrame and 
applying diffs to it.   

To avoid being tricked into fast-forwarding to an invalid state, the protocol 
ties Frames to the corresponding Blockchain by including Frame hashes in 
affiliated Block headers. A *FastForwardResponse* includes a Block and a Frame,
such that, upon receiving these objects, the requester may check the Frame hash
against the Block header, and count the Block signatures against the **known** 
set of validators, before resetting the Hashgraph from the Frame. 

Note the importance for the requester to be aware of the validaor set of the 
Hashgraph it wishes to sync with; this is how they can verify the Block 
signatures. With a dynamic validator set, an additional mechanism will be 
necessary to securely track changes to the validator set. 

Snapshot/Restore
----------------

It's one thing to catch-up with the Hashgraph and Blockchain, but nodes also
need to catch-up with the application state. we extended the Proxy interface 
with methods to retrieve and restore snapshots. 

::

  type AppProxy interface {
  	SubmitCh() chan []byte
  	CommitBlock(block hashgraph.Block) ([]byte, error)
  	GetSnapshot(blockIndex int) ([]byte, error)
  	Restore(snapshot []byte) error
  }

Snapshots are raw byte arrays, so it is up to the application layer to define 
what the snapshots represent, how they are encoded, and how they are used to 
restore the application to a particular state. The *GetSnapshot* method does
take a *blockIndex* int parameter, which implies that the application should 
somehow keep track of snapshots for every committed block. As the protocol 
evolves, we will likely link this to a *FrameRate* parameter to reduce the 
overload on the application caused by the need to take all these snapshots.

So together with a Frame and the corresponding Block, a FastForward request 
comes with a Snapshot of the application for the node to restore the application
to the corresponding state. If the Snapshot was incorrect, the node will 
immediately diverge from the main chain because it will obtain different state
hashes upon committing new blocks.

Improvements and Furter Work
----------------------------

The protocol is not entirely watertight yet; there are edge cases that could 
quickly lead to forks and diverging nodes. 

1) Events above the Frame that reference parents from before the Frame.
This is the cost of the Frame size vs content tradeoff.

2) The snapshot is not linked to the blockchain yet, only indirectly through
resulting StateHashes

Both these issues could be addressed with a general retry method. Make the 
FastForward method atomic, work on temporary copy of the Hashgraph, if a fork is
detected, try to FastSync again. This requires further work and policies on fork
detection and self-healing protocols.








