Usage
=====

In this section we will guide you through deploying an application on top of Babble.
Babble comes with the Dummy application which is used in this demonstration. It is a 
simple chat application where participants write messages on a channel and Babble
guarantees that everyone sees the same messages in the same order.

Docker
------

We have provided a series of scripts to bootstrap the demo. Let us first use the easy 
method to view the demo and then we will take a closer look at what is happening behind
the scenes.  

Make sure you have `Docker <https://docker.com>`__ installed.  

One of the scripts requires babble to be installed locally because it uses the  
**keygen** command to generate cryptographic key pairs in the format used by babble.  
To install babble:

::

    [...]/babble$ make install

Then, run the testnet:  

::

    [...]/babble$ cd docker
    [...]/babble/docker$ make


Once the testnet is started, a script is automatically launched to monitor consensus  
figures:  

::

    consensus_events:131055 consensus_transactions:0 events_per_second:265.53 id:0 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.24 sync_rate:1.00 transaction_pool:0 undetermined_events:26
    consensus_events:131055 consensus_transactions:0 events_per_second:266.39 id:3 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.34 sync_rate:1.00 transaction_pool:0 undetermined_events:25
    consensus_events:131055 consensus_transactions:0 events_per_second:267.30 id:2 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.44 sync_rate:1.00 transaction_pool:0 undetermined_events:31
    consensus_events:131067 consensus_transactions:0 events_per_second:268.27 id:1 last_consensus_round:14433 num_peers:3 round_events:11 rounds_per_second:29.54 sync_rate:1.00 transaction_pool:0 undetermined_events:21

Running ``docker ps -a`` will show you that 8 docker containers have been launched:  

::

    [...]/babble/docker$ docker ps -a
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

    [...]/babble/docker$ make demo

.. image:: assets/demo.png

Finally, stop the testnet:

::

    [...]/babble/docker$ make stop

Configuration
-------------

The first thing that the scripts do is to create the configuration files (cf ``build-conf.sh``)

Babble consumes a ``peers.json`` file that lists all the participants in the network.
Each participant is characterised by a public key and a network address.
Every node has a cryptographic key pair that is used to encrypt, sign and verify messages.
The private key is secret but the public key is used by other nodes to verify messages signed with the private key.
The encryption scheme used by Babble is ECDSA with the P256 curve.

Here is an example of a ``peers.json`` file defining a network of 4 participants:

::

    [
	{
		"NetAddr":"172.77.5.1:1337",
		"PubKeyHex":"0x042B6704A5414914B9C715ED21EBD34DDF70B900682F9F52F8DE5CB2F325884047B77238CC3827EB935FAC65290D901F39278BBB3CA7FB9AEFCFDE999ABDE5783F"
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

Babble Executable
-----------------

Let us take a look at the help provided by the Babble CLI:

::

    babble run --help
    
    NAME:
        main run - Run node

    USAGE:
        main run [command options] [arguments...]

    OPTIONS:
        --datadir value       Directory for the configuration (default: "/home/<usr>/.babble")
        --node_addr value     IP:Port to bind Babble (default: "127.0.0.1:1337")
        --no_client           Run Babble with dummy in-memory App client
        --proxy_addr value    IP:Port to bind Proxy Server (default: "127.0.0.1:1338")
        --client_addr value   IP:Port of Client App (default: "127.0.0.1:1339")
        --service_addr value  IP:Port of HTTP Service (default: "127.0.0.1:80")
        --log_level value     debug, info, warn, error, fatal, panic (default: "debug")
        --heartbeat value     Heartbeat timer milliseconds (time between gossips) (default: 1000)
        --max_pool value      Max number of pooled connections (default: 2)
        --tcp_timeout value   TCP timeout milliseconds (default: 1000)
        --cache_size value    Number of items in LRU caches (default: 500)
        --sync_limit value    Max number of events for sync (default: 1000)
	
    
Given this, it easier to understand what the rest of the scripts in the demo do. 

After packaging Babble and DummyApp in respective Docker containers, the ``run-testnet.sh`` script
is called to setup an overlay network and start Babble and DummyApp for each participants. Here is 
an extract: 

::

    for i in $(seq 1 $N)
    do
        docker create --name=node$i --net=babblenet --ip=172.77.5.$i babble run \
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

        docker run -d --name=client$i --net=babblenet --ip=172.77.5.$(($N+$i)) -it dummy \
        --name="client $i" \
        --client_addr="172.77.5.$(($N+$i)):1339" \
        --proxy_addr="172.77.5.$i:1338" \
        --log_level="info"
    done

Notice that the ``node_addr`` option corresponds to the address provided in the ``peers.json`` file.
Babble and the App are coupled by matching up their ``proxy_addr`` and ``client_addr`` settings.
Also important is that the ``peers.json`` file is copied to ``~/.babble`` which is the default directory
where Babble looks for configuration.

Stats and Logs
--------------

Once a node is up and running, we can call the ``Stats`` endpoint exposed by the http service:

::

    curl -s http://172.77.5.1:80/Stats
    

Or we can look at the logs produced by Babble:

::

    docker logs node1
