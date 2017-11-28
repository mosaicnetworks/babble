Design
=============

Overview
--------

::

            ========================================
            = APP                                  =  
            =                                      =
            =  ===============     ==============  =
            =  = Service     = <-- = State      =  =
            =  =             =     =            =  =
            =  ===============     ==============  =
            =          |                |          =
            =       ========================       =
            =       = Babble Proxy         =       =
            =       ========================       =
            =          |                ^          =
            ===========|================|===========
                       |                |
            --------- SubmitTx(tx) ---- CommitTx(tx) ------- (JSON-RPC/TCP)
                       |                |
         ==============|================|===============================
         = BABBLE      |                |                              =
         =             v                |                              =
         =          ========================                           =
         =          = App Proxy            =                           =
         =          =                      =                           =
         =          ========================                           = 
         =                     |                                       =
         =   =======================================                   =
         =   = Core                                =                   =
         =   =                                     =                   =
         =   =                                     =    ============   =
         =   =  =============        ===========   =    = Service  =   =  
         =   =  = Hashgraph =        = Store   =   = -- =          = <----> HTTP API
         =   =  =============        ===========   =    =          =   =
         =   =                                     =    ============   =     
         =   =                                     =                   =
         =   =======================================                   =
         =                     |                                       =
         =   =======================================                   =
         =   = Transport                           =                   =
         =   =                                     =                   =
         =   =======================================                   =
         =                     ^                                       =
         ======================|========================================
                               |                  
                               v
                       
                            Network

The above diagram shows how Babble fits in the typical architecture of a distributed  
application. Users interact with an App's Service which reads data from its State.  
However, the Service never updates the State directly. Instead, it passes commands  
to an ordering system which communicates to other nodes and feeds the commands back  
to the State in consensus order. Babble is an ordering system that plugs into the  
App thanks to a very simple JSON-RPC interface over TCP.

Proxy
-----

The Babble node and the App are loosely coupled and can run in separate processes.  
They communicate via a very simple **JSON-RPC** interface over a **TCP** connection. 

The App submits transactions for consensus ordering via the **SubmitTx** endpoint  
exposed by the **App Proxy**. Babble asynchrously processes transactions and  
eventually feeds them back to the App, in consensus order,  with a **CommitTx**  
message.

Transactions are just raw bytes and Babble does not need to know what  
they represent. Therefore, encoding and decoding transactions is done by the App.

Apps must implement their own **Babble Proxy** to submit and receive committed  
transactions from Babble. The advantage of using a JSON-RPC API is that there is  
no restriction on the programming language for the App. It only requires a component    
that sends SubmitTx messages to Babble and exposes a TCP enpoint where Babble can  
send CommitTx messages.

When launching a Babble node, one must specify the address and port exposed by the  
Babble Proxy of the App. It is also possible to configure which address and port  
the App Proxy exposes.

Example SubmitTx request (from App to Babble):

::

    request: {"method":"Babble.SubmitTx","params":["Y2xpZW50IDE6IGhlbGxv"],"id":0}
    response: {"id":0,"result":true,"error":null}


Note that the Proxy API is **not** over HTTP; It is raw JSON over TCP. Here is an  
example of how to make a SubmitTx request manually:  

::

    printf "{\"method\":\"Babble.SubmitTx\",\"params\":[\"Y2xpZW50IDE6IGhlbGxv\"],\"id\":0}" | nc -v  172.77.5.1 1338


Example CommitTx request (from Babble to App):

::

    request: {"method":"State.CommitTx","params":["Y2xpZW50IDE6IGhlbGxv"],"id":0}
    response: {"id":0,"result":true,"error":null}

The content of "params" is the base64 encoding of the raw transaction bytes ("client1: hello").

Transport
---------

Babble nodes communicate with other Babble nodes in a fully connected Peer To Peer  
network. Nodes gossip by repeatedly choosing another node at random and telling  
eachother what they know about the  Hashgraph. The gossip protocol is extremely  
simple and serves the dual purpose of gossiping about transactions and about the  
gossip itself (the Hashgraph). The Hashraph contains enough information to compute  
a consensus ordering of transactions. 

The communication mechanism is a custom RPC protocol over TCP connections. It  
implements a Pull-Push gossip system. At the moment, there are two types of RPC  
commands: **Sync** and **EagerSync**. When node **A** wants to sync with node **B**,  
it sends a **SyncRequest** to **B** containing a description of what it knows  
about the Hashgraph. **B** computes what it knows that **A** doesn't know and  
returns a **SyncResponse** with the corresponding events in topological order.  
Upon receiving the **SyncResponse**, **A** updates its Hashgraph accordingly and  
calculates the consensus order. Then, **A** sends an **EagerSyncRequest** to **B**  
with the Events that it knowns and **B** doesn't. Upon receiving the **EagerSyncRequest**,  
**B** updates its Hashgraph and runs the consensus methods.

The list of peers must be predefined and known to all peers. At the moment, it is  
not possible to dynamically modify the list of peers while the network is running  
but this is not a limitation of the Hashgraph algorithm, just an implemention  
prioritization.

Core
----

The core of Babble is the component that maintains and computes the Hashgraph.  
The consensus algorithm, invented by Leemon Baird, is best described in the `white-paper <http://www.swirlds.com/downloads/SWIRLDS-TR-2016-01.pdf>`__  
and its `accompanying document <http://www.swirlds.com/downloads/SWIRLDS-TR-2016-02.pdf>`__. 

The Hashgraph itself is a data structure that contains all the information about  
the history of the gossip and thereby grows and grows in size as gossip spreads.  
There are various strategies to keep the size of the Hashgraph limited. In our  
implementation, the **Hashgraph** object has a dependency on a **Store** object  
which contains the actual data and is abstracted behind an interface.

There are currently two implementations of the **Store** interface. The ``InmemStore``
uses a set of in-memory LRU caches which can be extended to persist stale items 
to disk and the size of the LRU caches is configurable. The ``BadgerStore`` is 
a wrapper around this cache that also persists objects to a key-value store on disk.
The database produced by the ``BadgerStore`` can be reused to bootstrap a node 
back to a specific state.

Service
-------

The Service exposes an HTTP API to query information about a node. At the  
moment, it only exposes a **Stats** endpoint:  

::

    $curl -s http://[ip]:8080/Stats | jq
    {
      "consensus_events": "199993",
      "consensus_transactions": "0",
      "events_per_second": "264.65",
      "id": "0",
      "last_consensus_round": "21999",
      "num_peers": "3",
      "round_events": "10",
      "rounds_per_second": "29.11",
      "sync_rate": "1.00",
      "transaction_pool": "0",
      "undetermined_events": "24",
      "state": "Babbling",
    }


