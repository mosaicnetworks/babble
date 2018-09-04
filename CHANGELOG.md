## UNRELEASED

SECURITY:

FEATURES:

IMPROVEMENTS:

BUG FIXES:

## v0.3.0 (September 4, 2018)

FEATURES:

* hashgraph: Replaced Leemon Baird's original "Fair" ordering method with 
Lamport timestamps.
* hashgraph: Introduced the concept of Frames and Roots to enable initializing a
hashgraph from a "non-zero" state.
* node: Added FastSync protocol to enable nodes to catch up with other nodes 
without downloading the entire hashgraph. 
* proxy: Introduce Snapshot/Restore functionality.

IMPROVEMENTS:

* hashgraph: Refactored the consensus methods around the concept of Frames.
* hashgraph: Removed special case for "initial" Events, and make use of Roots 
instead. 
* docs: Added sections on Babble and FastSync.