.. _introduction:

Introduction
============

What is Babble?
---------------

Babble is a software component that allows many computers to behave as one; a 
technique known as state machine replication. This can be useful to augment the 
fault tolerance of a service. It also empowers applications to embrace a fully 
distributed architecture where computation is no longer carried on a single, 
centralized, server.

Babble is designed to easily plug into applications written in any programming 
language. Developers can focus on building the application logic and simply 
integrate with Babble to handle the replication aspect. Basically, Babble will 
connect to other Babble nodes and guarantee that everyone processes the same 
commands in the same order. To do this, it uses Peer to Peer (P2P) networking 
and a Byzantine Fault Tolerant (BFT) consensus algorithm which brings together 
many appealing properties, like speed, asyncronisity and leaderlessness, of 
which other algorithms only implement subsets.  
