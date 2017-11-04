Introduction
============

What is Babble?
---------------

Babble is a software component that allows many computers to behave as one; a technique known as state machine replication.
This can be useful to augment the fault tolerance of a service. It also empowers applications to embrace a fully distributed
architecture where computation is no longer carried on a single, centralized, server.

Babble is designed to easily plug into applications written in any programming language. Developers can focus on building the
application logic and simply integrate with Babble to handle the replication aspect. Basically, Babble will connect to other Babble
nodes and guarantee that everyone processes the same commands in the same order. To do this, it uses Peer to Peer (P2P) networking and
a consensus algorithm called Hashgraph. This state of the art algorithm tolerates machines failing arbitrarily, including malicious 
behaviour; an ability known as Byzantine Fault Tolerance (BFT). Among the familly of BFT algorithms, Hashgraph brings together many
appealing properties of which other algorithms can only implement subsets.  
