.. _fastsync:

FastSync
========

FastSync is a part of the Babble protocol which enables nodes to catch up with
other nodes without having to download the entire history of gossip (Hashgraph + 
Blockchain). This is important in the context of mobile ad hoc networks where 
users create or join groups dynamically and quickly, and where limited computing
resources demand periodic pruning of the underlying data store. The solution 
requires nodes to regularly capture snapshots of the application state, and to 
link each snapshot to the section of the Hashgraph it resulted from. A node that 
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
coinciding block. With this information, and having verified the block 
signatures against the other items and the validator set, the requesting node 
attempts to reset its Hashgraph from the Frame, and restore the application from 
the snapshot. The difficulty resides in defining what we mean by 
*last consensus* snapshot, and how to package enough information in the Frames 
as to form a base for a new/pruned Hashgraph. 

Frames
------

Frames are self-contained sections of the Hashgraph. They are composed of Roots 
and regular Hashgraph Events, where Roots are the base on top of which Events 
can be inserted. Basically, given a Frame, we can initialize a new Hashgraph and 
continue gossiping on top of it; earlier records of the gossip history are 
discarded/pruned. 

A Frame corresponds to a Hashgraph consensus round. Indeed, the consensus 
algorithm commits Events in batches, which we map onto a Frame, and finally onto 
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
fine as long as we do not expect to initialize Hashgraphs 'on the go'.

We introduced the concept of Roots, to remove the special case and handle more
general situations, allowing to initialize a Hashgraph from a section of an 
existing Hashgraph.

.. image:: assets/roots_1.png

The new rule prescribes that an Event should only be inserted if its parents 
belong to the Hashgraph or are referenced in one of the Roots. 

[Detail of what a Root actually contains] => not entirely sure yet. Fix the code
before finishing this section.

Transition _ could still fail if there are undetermined events below the Frame.
why?

Reset
-----

Resetting a Hashgraph from a Frame

Importance of Agreeing on Roots (need to be signed somehow) => Frame, FrameHash,
Block signatures

Reseting can fail if there were undecided Events below the Frame

AnchorBlock
-----------

Collecting signatures, Importance of Blockchain mapping

FrameRate?

State Snapshot Interface
------------------------

Snapshot / Restore

'Loose' protocol

Verification
------------

FrameHash + Snapshot + StateHash

Counting signatures







