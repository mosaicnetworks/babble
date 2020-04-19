.. _usage:

Usage
=====

In this section we will guide you through deploying an application on top of
Babble using the socket interface. Babble comes with the Dummy application which
is used in this demonstration. It is a simple chat application where users 
write messages on a channel and Babble guarantees that everyone sees the same
messages in the same order.

Docker
------

We have provided a series of scripts to bootstrap a demo. Let us first use the
easy method to view the demo and then we will take a closer look at what is
happening behind the scenes.

Make sure you have `Docker <https://docker.com>`__ installed.

The demo will pull Docker images from our `official public Docker registry
<https://hub.docker.com/u/mosaicnetworks/>`__

.. code:: bash

    [...]/babble$ cd demo
    [...]/babble/demo$ make


Once the testnet is started, a script is automatically launched to monitor
consensus figures:

::

    consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:1 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:18
    consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:3 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:20
    consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:2 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:21
    consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:0 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:20

Running ``docker ps -a`` will show you that 9 docker containers have
been launched:

.. code:: bash

    [...]/babble/demo$ docker ps -a
    CONTAINER ID        IMAGE                    COMMAND                  CREATED             STATUS              PORTS                   NAMES
    ba80ef275f22        mosaicnetworks/watcher   "/watch.sh"              48 seconds ago      Up 7 seconds                                watcher
    4620ed62a67d        mosaicnetworks/dummy     "dummy '--name=client"   49 seconds ago      Up 48 seconds       1339/tcp                client4
    847ea77bd7fc        mosaicnetworks/babble    "babble run --cache_s"   50 seconds ago      Up 49 seconds       80/tcp, 1337-1338/tcp   node4
    11df03bf9690        mosaicnetworks/dummy     "dummy '--name=client"   51 seconds ago      Up 50 seconds       1339/tcp                client3
    00af002747ca        mosaicnetworks/babble    "babble run --cache_s"   52 seconds ago      Up 50 seconds       80/tcp, 1337-1338/tcp   node3
    b2011d3d65bb        mosaicnetworks/dummy     "dummy '--name=client"   53 seconds ago      Up 51 seconds       1339/tcp                client2
    e953b50bc1db        mosaicnetworks/babble    "babble run --cache_s"   53 seconds ago      Up 52 seconds       80/tcp, 1337-1338/tcp   node2
    0c9dd65de193        mosaicnetworks/dummy     "dummy '--name=client"   54 seconds ago      Up 53 seconds       1339/tcp                client1
    d1f4e5008d4d        mosaicnetworks/babble    "babble run --cache_s"   55 seconds ago      Up 54 seconds       80/tcp, 1337-1338/tcp   node1


Indeed, each node is comprised of an App and a Babble node (cf Design section).
The ``watcher`` container monitors consensus figures.

Run the ``demo`` script to play with the ``Dummy App`` which is a simple chat
application powered by the Babble consensus platform:

.. code:: bash

    [...]/babble/demo$ make demo

.. image:: assets/demo.png

Finally, stop the testnet:

.. code:: bash

    [...]/babble/demo$ make stop

Manual Setup
------------

The above scripts hide a lot of the work required for setting up a Babble
network. They generate the configuration files automatically, copy them to the
right places and launch the nodes in Docker containers. We recommend looking at
these scripts closely to understand how the demo works. Here, we will attempt
to explain the individual steps that take place behind the scenes.

Configuration
~~~~~~~~~~~~~

Babble reads configuration from the directory specified by the ``datadir`` flag
which defaults to ``~/.babble`` on UNIX systems. This directory should contain
the following files:

 - ``priv_key``    : The private key of the validator runnning the node.
 - ``peers.json``  : The current validator-set.
 - ``genesis.peers.json`` : (optional, default peers.json) The initial
   validator-set of the network.

Keys
****

Every participant has a cryptographic key-pair that it uses to encrypt, sign
and verify messages. The private key is secret but the public key is used by
other nodes to verify messages signed with the private key.

The encryption scheme used by Babble is ECDSA with the secp256k1 curve (like
Bitcoin and Ethereum).

Babble's ``keygen`` command can be used to generate key-pairs in the
appropriate format.

Peers
*****

``peers.json`` and ``genesis.peers.json`` are used to determine the current and
initial validator-sets of a network.

``genesis.peers.json`` corresponds to the initial validator-set; the one that
the hashgraph was started with. If ``genesis.peers.json`` is not provided,
Babble will use ``peers.json`` as the genesis validator-set.


``peers.json`` corresponds to the current validator-set. These are the nodes
that are allowed to record new Events in the hashgraph, and who will gossip
among each other.

``peers.json`` and ``gensesis.peers.json`` are not necessarily equal because
the :ref:`dynamic membership protocol <dynamic_membership>` enables new nodes to
join or leave a live Babble network dynamically. It is important for a joining
node to know the initial validator-set in order to replay and verify the
hashgraph up to the point where it joins.

It is possible to start a Babble network with just a single node, or with a
predefined validator-set composed of multiple nodes.

In the latter case, someone, or some process, needs to aggregate the public
keys and network addresses of all participants into a single file
(``peers.genesisjson``), and ensure that everyone has a copy of this file. It is
left to the user to derive a scheme to produce the configuration files but the
docker demo scripts are a good place to start.

Example
*******

Let us say I want to participate in a Babble network. I am going to start by
running ``babble keygen`` to create a key-pair:

.. code:: bash

  babble keygen
  Your private key has been saved to: /home/[user]/.babble/priv_key
  Your public key has been saved to: /home/[user]/.babble/key.pub

Next, I am going to copy the public key (key.pub) and communicate it to whoever
is responsible for producing the peers.json file. At the same time, I will tell
them that I am going to be listening on 172.77.5.2:1337. You may also optionally
supply a moniker for each node, which is far more readable than a
public key address.

Suppose three other people do the same thing. The resulting peers.json file
could look something like this:

.. code:: json

    [
      {
         "NetAddr":"172.77.5.1:1337",
         "PubKeyHex":"0x0471AEE3CAE4E8442D37C9F5481FB32C4531511988652DF923B79ED4ED992021183D31E0F6FBFE96D89B6D03D7250292DFECD4FC414D83A5C38FA3FAD0D8572864",
         "Moniker":"node1"
      },
      {
         "NetAddr":"172.77.5.2:1337",
         "PubKeyHex":"0x045E034D73C849756AE7B6515CA60D96A5A911B13A4D8B45BC0E0B02EDB45009DF6CCC074EEB6F7C6795740F993664EDEE970F8A717C89344F8437F412BDF0D17C",
         "Moniker":"node2"
      },
      {
         "NetAddr":"172.77.5.3:1337",
         "PubKeyHex":"0x047CCCD40D90B331C64CE27911D3A31AF7DC16C1EA6D570FDC2120920663E0A678D7B5D0C19B6A77FEA829F8198F4F487B68206B93B7AD17D7C49CA7E0164D0033",
         "Moniker":"node3"
      },
      {
         "NetAddr":"172.77.5.4:1337",
         "PubKeyHex":"0x0406CB5043E7337700E3B154993C872B1C61A84B1A739528C4A10135A3D64939C094B4A999BD21C3D5E9E9ECF15B202414F073795C9483B2F51ADA7EE59EB5EAC4",
         "Moniker":"node4"
      }
    ]

Now everyone is going to take a copy of this peers.json file and put it in a
folder together with the priv_key file they generated in the previous step.
That is the folder that they need to specify as the datadir when they run
Babble.

Babble Executable
-----------------

Let us take a look at the help provided by the Babble CLI:

.. code:: bash

    Run node

    Usage:
      babble run [flags]
    
    Flags:
      -a, --advertise string          Advertise IP:Port for babble node
          --bootstrap                 Load from database
          --cache-size int            Number of items in LRU caches (default 10000)
      -c, --client-connect string     IP:Port to connect to client (default "127.0.0.1:1339")
          --datadir string            Top-level directory for configuration and data (default "/home/martin/.babble")
          --db string                 Dabatabase directory (default "/home/martin/.babble/badger_db")
          --fast-sync                 Enable FastSync
          --heartbeat duration        Timer frequency when there is something to gossip about (default 10ms)
      -h, --help                      help for run
      -j, --join-timeout duration     Join Timeout (default 10s)
      -l, --listen string             Listen IP:Port for babble node (default "127.0.0.1:1337")
          --log string                debug, info, warn, error, fatal, panic (default "debug")
      -R, --maintenance-mode          Start Babble in a suspended (non-gossipping) state
          --max-pool int              Connection pool size max (default 2)
          --moniker string            Optional name
          --no-service                Disable HTTP service
      -p, --proxy-listen string       Listen IP:Port for babble proxy (default "127.0.0.1:1338")
      -s, --service-listen string     Listen IP:Port for HTTP service (default "127.0.0.1:8000")
          --signal-addr string        IP:Port of WebRTC signaling server (default "127.0.0.1:2443")
          --signal-skip-verify        (Insecure) Accept any certificate presented by the signal server
          --slow-heartbeat duration   Timer frequency when there is nothing to gossip about (default 1s)
          --store                     Use badgerDB instead of in-mem DB
          --suspend-limit int         Limit of undetermined events (per node) before entering suspended state (default 100)
          --sync-limit int            Max number of events for sync (default 1000)
      -t, --timeout duration          TCP Timeout (default 1s)
          --webrtc                    Use WebRTC transport
    
    

The ``listen`` flag controls the local address:port where this node gossips with
other nodes. If the node is running behind some kind of NAT, it is possilbe to
advertise a different address with the ``advertise`` flag. If ``advertise`` is 
not  specified, the node defaults to using the ``listen`` address. By default 
``listen`` is ``127.0.0.1:1337``, meaning that Babble will bind to the loopback
addresse on the local machine.

As we explained in the architecture section, each Babble node works in
conjunction with an application for which it orders transactions. When Babble
and the application are connected by a TCP interface, we specify two other
endpoints:

 - ``proxy-listen``  : where Babble listens for transactions from the App
 - ``client-connect`` : where the App listens for transactions from Babble

We can also specify where Babble exposes its HTTP API providing information on
the Hashgraph and Blockchain data store. This is controlled by the optional
``service-listen`` flag.

The ``fast-sync`` parameter determines whether or not the node will attempt to
fast-forward to the tip of the hashgraph, or download and replay the entire
hashgraph from start. More on this in :ref:`fast-sync <fastsync>`

We can choose to run Babble with a database backend or only with an in-memory 
cache. With the ``store`` flag set, Babble will look for a database file in
``datadir``/babdger_db or in the path specified by ``db``. If the database 
already exists, and the ``--boostrap`` flag is set, the node will load the 
database and bootstrap itself to a state consistent with the database and it
will be able to proceed with the consensus algorithm from there. If the database
does not exist yet, or the ``--bootstrap`` flag is not set, a new one will be
created and the node will start from a clean state.

The node can also be started in ``maintenance-mode`` with the homonymous flag. 
The node is started normally but goes straight into the ``Suspended`` state,
where it still  responds to sync-requests, and service API requests, but does
not produce or insert new Events in the underlying hashgraph. The ``Suspended``
state is also triggered automatically when more than ``suspend-limit`` 
undetermined-events were created since last starting the node. This is a 
safeguard against runaway conditions when a network does not have a strong 
majority and produces undetermined-events ad infinitum.   

Here is how the Docker demo starts Babble nodes together wth the Dummy
application:

.. code:: bash

    for i in $(seq 1 $N)
    do
        docker run -d --name=client$i --net=babblenet --ip=172.77.10.$i -it mosaicnetworks/dummy:latest \
        --name="client $i" \
        --client-listen="172.77.10.$i:1339" \
        --proxy-connect="172.77.5.$i:1338" \
        --discard \
        --log="debug"
    done

    for i in $(seq 1 $N)
    do
        docker create --name=node$i --net=babblenet --ip=172.77.5.$i mosaicnetworks/babble:latest run \
        --heartbeat=100ms \
        --moniker="node$i" \
        --cache-size=50000 \
        --listen="172.77.5.$i:1337" \
        --proxy-listen="172.77.5.$i:1338" \
        --client-connect="172.77.10.$i:1339" \
        --service-listen="172.77.5.$i:80" \
        --sync-limit=500 \
        --fast-sync=$FASTSYNC \
        --store \
        --log="debug"

        docker cp $MPWD/conf/node$i node$i:/.babble
        docker start node$i
    done

Stats, blocks and Logs
----------------------

Once a node is up and running, we can call the ``stats`` endpoint exposed
by the HTTP service:

.. code:: bash

    curl -s http://172.77.5.1:80/stats

Or request to see a specific block:

.. code:: bash

    curl -s http://172.77.5.1:80/block/1

Or we can look at the logs produced by Babble:

.. code:: bash

    docker logs node1

We can look at the current state of docker containers:

.. code:: bash

    docker ps --all
