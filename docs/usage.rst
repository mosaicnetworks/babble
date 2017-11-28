Usage
=====

In this section we will guide you through deploying an application on top of Babble.
Babble comes with the Dummy application which is used in this demonstration. It is a 
simple chat application where participants write messages on a channel and Babble
guarantees that everyone sees the same messages in the same order.

Docker
------

We have provided a series of scripts to bootstrap a demo. Let us first use the easy 
method to view the demo and then we will take a closer look at what is happening behind
the scenes.  

Make sure you have `Docker <https://docker.com>`__ installed.  

The demo will pull Docker images from our `official public Docker registry <https://hub.docker.com/u/mosaicnetworks/>`__ 

::

    [...]/babble$ cd demo
    [...]/babble/demo$ make


Once the testnet is started, a script is automatically launched to monitor consensus  
figures:  

::

    consensus_events:131055 consensus_transactions:0 events_per_second:265.53 id:0 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.24 sync_rate:1.00 transaction_pool:0 undetermined_events:26
    consensus_events:131055 consensus_transactions:0 events_per_second:266.39 id:3 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.34 sync_rate:1.00 transaction_pool:0 undetermined_events:25
    consensus_events:131055 consensus_transactions:0 events_per_second:267.30 id:2 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.44 sync_rate:1.00 transaction_pool:0 undetermined_events:31
    consensus_events:131067 consensus_transactions:0 events_per_second:268.27 id:1 last_consensus_round:14433 num_peers:3 round_events:11 rounds_per_second:29.54 sync_rate:1.00 transaction_pool:0 undetermined_events:21

Running ``docker ps -a`` will show you that 8 docker containers have been launched:  

::

    [...]/babble/demo$ docker ps -a
    CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS               NAMES
    9e4c863c9e83        dummy               "dummy '--name=cli..."   9 seconds ago       Up 8 seconds        1339/tcp            client4
    40c92938a986        babble              "babble run --cach..."   10 seconds ago      Up 9 seconds        1337-1338/tcp       node4
    4c1eb201d29d        dummy               "dummy '--name=cli..."   10 seconds ago      Up 9 seconds        1339/tcp            client3
    cda62387e4fd        babble              "babble run --cach..."   11 seconds ago      Up 9 seconds        1337-1338/tcp       node3
    765c73d66bcf        dummy               "dummy '--name=cli..."   11 seconds ago      Up 10 seconds       1339/tcp            client2
    a6895aaa141a        babble              "babble run --cach..."   11 seconds ago      Up 10 seconds       1337-1338/tcp       node2
    8f996e13eda7        dummy               "dummy '--name=cli..."   12 seconds ago      Up 11 seconds       1339/tcp            client1
    36c97a968d22        babble              "babble run --cach..."   12 seconds ago      Up 11 seconds       1337-1338/tcp       node1

Indeed, each node is comprised of an App and a Babble node (cf Design section).

Run the ``demo`` script to play with the ``Dummy App`` which is a simple chat application
powered by the Babble consensus platform:

::

    [...]/babble/demo$ make demo

.. image:: assets/demo.png

Finally, stop the testnet:

::

    [...]/babble/demo$ make stop

Manual Setup
------------

The above scripts hide a lot of the complications involved in setting up a Babble network.
They generate the configuration files automatically, copy them to the right places and launch
the nodes in Docker containers. We recommend looking at these scripts closely to understand 
how the demo works. Here, we will attempt to explain the individual steps that take place 
behind the scenes.

Configuration 
~~~~~~~~~~~~~

Babble reads configuration from the directory specified by the ``datadir`` flag which defaults
to ``~/.babble`` on linux/osx. This directory must contain two files:

 - ``peers.json``  : Lists all the participants in the network.
 - ``priv_key.pem``: Contains the private key of the validator runnning the node. 

Every participant has a cryptographic key-pair that is used to encrypt, sign and verify messages. 
The private key is secret but the public key is used by other nodes to verify messages signed with
the private key. The encryption scheme used by Babble is ECDSA with the P256 curve.

To run a Babble network, it is necessary to predefine who the participants are going to be. Each
participant will generate a key-pair and decide which network address it is going to be using for 
the Babble protocol. Someone, or some process, then needs to aggregate the public keys and network
addresses of all participants into a single file - the peers.json file. Every participant uses a 
copy of the same peers.json file. Babble will read that file to identify the participants in the 
network, communicate with them and verify their cryptographic signatures.

To generate key-pairs in a format usable by Babble, we have created the ``keygen`` command. It
is left to the user to derive a scheme to produce the configuration files but the docker demo
scripts are a good place to start.

So let us say I want to participate in a Babble network. I am going to start by running ```babble keygen```
to create a key-pair:

::

    babble keygen
    PublicKey:
    0x0471AEE3CAE4E8442D37C9F5481FB32C4531511988652DF923B79ED4ED992021183D31E0F6FBFE96D89B6D03D7250292DFECD4FC414D83A5C38FA3FAD0D8572864
    PrivateKey:
    -----BEGIN EC PRIVATE KEY-----
    MHcCAQEEIP53jgf9lul6mvJvT8835DoDNRy7t8oIu4fU2mq2hn7ToAoGCCqGSM49
    AwEHoUQDQgAEca7jyuToRC03yfVIH7MsRTFRGYhlLfkjt57U7ZkgIRg9MeD2+/6W
    2JttA9clApLf7NT8QU2DpcOPo/rQ2FcoZA==
    -----END EC PRIVATE KEY-----
	

Then, I am going to copy the private key and paste it in a priv_key.pem file. So the content of the
priv_key.pem file would be:

::

    -----BEGIN EC PRIVATE KEY-----
    MHcCAQEEIP53jgf9lul6mvJvT8835DoDNRy7t8oIu4fU2mq2hn7ToAoGCCqGSM49
    AwEHoUQDQgAEca7jyuToRC03yfVIH7MsRTFRGYhlLfkjt57U7ZkgIRg9MeD2+/6W
    2JttA9clApLf7NT8QU2DpcOPo/rQ2FcoZA==
    -----END EC PRIVATE KEY-----


Next, I am going to copy the public key (0x0471AEE3C...) and communicate it to whoever is responsible
for producing the peers.json file. At the same time, I will tell them that I am going to be listening 
on 172.77.5.2:1337.

Suppose three other people do the same thing. The resulting peers.json file could look something like this:

::

    [
	{
		"NetAddr":"172.77.5.1:1337",
		"PubKeyHex":"0x0471AEE3CAE4E8442D37C9F5481FB32C4531511988652DF923B79ED4ED992021183D31E0F6FBFE96D89B6D03D7250292DFECD4FC414D83A5C38FA3FAD0D8572864"
	},
	{
		"NetAddr":"172.77.5.2:1337",
		"PubKeyHex":"0x0448E914D5704E9018FCF1B142E63D1E7BFEE8C81C8E9285D98742671FDDE65F0C0C43A42A02BBE8ADE3DCA0A7C43B7EADA97DC58D2B907FEA2C8F26132D0CF63B"
	},
	{
		"NetAddr":"172.77.5.3:1337",
		"PubKeyHex":"0x047CCCD40D90B331C64CE27911D3A31AF7DC16C1EA6D570FDC2120920663E0A678D7B5D0C19B6A77FEA829F8198F4F487B68206B93B7AD17D7C49CA7E0164D0033"
	},
	{
		"NetAddr":"172.77.5.4:1337",
		"PubKeyHex":"0x0406CB5043E7337700E3B154993C872B1C61A84B1A739528C4A10135A3D64939C094B4A999BD21C3D5E9E9ECF15B202414F073795C9483B2F51ADA7EE59EB5EAC4"
	}
    ]

Now everyone is going to take a copy of this peers.json file and put it in a folder together with the
priv_key.pem file they generated in the previous step. That is the folder that they need to specify as
the datadir when they run Babble.

Babble Executable
-----------------

Let us take a look at the help provided by the Babble CLI:

::

    NAME:
    babble run - Run node

    USAGE:
    babble run [command options] [arguments...]

    OPTIONS:
       --datadir value       Directory for the configuration (default: "/home/martin/.babble")
       --node_addr value     IP:Port to bind Babble (default: "127.0.0.1:1337")
       --no_client           Run Babble with dummy in-memory App client
       --proxy_addr value    IP:Port to bind Proxy Server (default: "127.0.0.1:1338")
       --client_addr value   IP:Port of Client App (default: "127.0.0.1:1339")
       --service_addr value  IP:Port of HTTP Service (default: "127.0.0.1:8000")
       --log_level value     debug, info, warn, error, fatal, panic (default: "debug")
       --heartbeat value     Heartbeat timer milliseconds (time between gossips) (default: 1000)
       --max_pool value      Max number of pooled connections (default: 2)
       --tcp_timeout value   TCP timeout milliseconds (default: 1000)
       --cache_size value    Number of items in LRU caches (default: 500)
       --sync_limit value    Max number of events for sync (default: 1000)
       --store value         badger, inmem (default: "badger")
       --store_path value    File containing the store database (default: "/home/martin/.babble/badger_db")

	
So we have just seen what the ``datadir`` flag does. The ``node_addr`` flag corresponds to the NetAddr
in the peers.json file; that is the endpoint that Babble uses to communicate with other Babble nodes.

As we explained in the architecture section, each Babble node works in conjunction with an application for
which it orders transactions. Babble and the application are connected by a TCP interface. Therefore, we need
to specify two other endpoints:

 - ``proxy_addr``  : where Babble listens for transactions from the App
 - ``client_addr`` : where the App listens for transactions from Babble 

We also need to specify where Babble exposes its HTTP API where one can query the Hashgraph data store.
This is defined by the ``service_addr`` flag.

Finally, we can choose to run Babble with a database backend or only with an in-memory
cache. By default, Babble will look for a database file in ``~/.babble/babdger_db``
but this can be set with the ``store_path`` option. If the file exists, the node
will load the database and bootstrap itself to a state consistent with the database 
and it will be able to proceed with the consensus algorithm from there. If the 
file does not exist yet, it will be created and the node will start from a clean slate. 

In some cases, it can be preferable to run Babble without a database backend. Indeed,
even if using a database can be indispensable in some deployments, it has a big 
impact on performance. To use an in-memory store only, set the option ``store inmem``.


Here is how the Docker demo starts Babble nodes together wth the Dummy application:

::

    for i in $(seq 1 $N)
    do
        docker create --name=node$i --net=babblenet --ip=172.77.5.$i mosaicnetworks/babble run \
        --cache_size=50000 \
        --tcp_timeout=200 \
        --heartbeat=10 \
        --node_addr="172.77.5.$i:1337" \
        --proxy_addr="172.77.5.$i:1338" \
        --client_addr="172.77.5.$(($N+$i)):1339" \
        --service_addr="172.77.5.$i:80" \
        --sync_limit=500
        docker cp $MPWD/conf/node$i node$i:/.babble
        docker start node$i

        docker run -d --name=client$i --net=babblenet --ip=172.77.5.$(($N+$i)) -it mosaicnetworks/dummy \
        --name="client $i" \
        --client_addr="172.77.5.$(($N+$i)):1339" \
        --proxy_addr="172.77.5.$i:1338" \
        --log_level="info"
    done

Stats and Logs
--------------

Once a node is up and running, we can call the ``Stats`` endpoint exposed by the HTTP service:

::

    curl -s http://172.77.5.1:80/Stats
    

Or we can look at the logs produced by Babble:

::

    docker logs node1
