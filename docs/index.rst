.. _index:

.. toctree::
   :hidden:
   :maxdepth: 2

   self
   design.rst
   api.rst
   config.rst
   install.rst
   usage.rst
   consensus.rst
   blockchain.rst
   state_machine.rst
   fastsync.rst
   dynamic_membership.rst


.. _introduction:

Introduction
============

What is Babble?
---------------

Babble is a distributed consensus engine designed to easily plug into any
application. Developers of distributed applications can focus on building their
application logic whilst Babble takes care of synchonization. Babble will 
connect to other Babble nodes and guarantee that everyone processes the same 
commands in the same order. To do this, it uses p2p networking and a Byzantine 
Fault Tolerant (BFT) consensus algorithm.

.. image:: assets/babble-network.png
   :height: 453px
   :width: 640px
   :align: center

Babble has the following features:

- **Asynchronous**:
    Participants have the freedom to process commands at different times.

- **Leaderless**:
    No participant plays a 'special' role.

- **Byzantine Fault-Tolerant**:
    Supports one third of faulty nodes, including malicious behavior.

- **Finality**:
    Babble's output can be used immediately, no need for block confirmations,
    etc.

- **Dynamic Membership**:
    Members can join or leave a Babble network without undermining security.

- **Fast Sync**:
    Joining nodes can sync straight to the current state of a network.

- **Accountability**: 
    Auditable history of the consensus algorithm's output.

- **Language Agnostic**: 
    Integrate with applications written in any programming language.

- **Mobile**:
    Bindings for Android and iOS.

- **WebRTC**:
    Support for WebRTC connections for practical p2p connections.
