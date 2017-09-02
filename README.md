# BABBLE
## Consensus platform for distributed applications.  

Nodes in a distributed application require a component to communicate transactions  
between all participants before processing them locally in a consistent order.  
Babble is a plug and play solution for this component.  It uses the Hashgraph   
consensus algorithm which offers definitive advantages over other BFT systems.

## Architecture
```
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
```
The above diagram shows how Babble fits in the typical architecture of a distributed  
application. Users interact with an App's Service which reads data from its State.  
However, the Service never updates the State directly. Instead, it passes commands  
to an ordering system which communicates to other nodes and feeds the commands back  
to the State in consensus order. Babble is an ordering system that plugs into the  
App thanks to a very simple JSON-RPC interface over TCP.

### Proxy

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
```json
request: {"method":"Babble.SubmitTx","params":["Y2xpZW50IDE6IGhlbGxv"],"id":0}
response: {"id":0,"result":true,"error":null}
```

Note that the Proxy API is **not** over HTTP; It is raw JSON over TCP. Here is an  
example of how to make a SubmitTx request manually:  
```bash
printf "{\"method\":\"Babble.SubmitTx\",\"params\":[\"Y2xpZW50IDE6IGhlbGxv\"],\"id\":0}" | nc -v  172.77.5.1 1338
```

Example CommitTx request (from Babble to App):
```json
request: {"method":"State.CommitTx","params":["Y2xpZW50IDE6IGhlbGxv"],"id":0}
response: {"id":0,"result":true,"error":null}
```
The content of "params" is the base64 encoding of the raw transaction bytes ("client1: hello").

### Transport

Babble nodes communicate with other Babble nodes in a fully connected Peer To Peer  
network. Nodes gossip by repeatedly choosing another node at random and telling  
eachother what they know about the  Hashgraph. The gossip protocol is extremely  
simple and serves the dual purpose of gossiping about transactions and about the  
gossip itself (the Hashgraph). The Hashraph contains enough information to compute  
a consensus ordering of transactions. 

The communication mechanism is a custom RPC protocol over TCP connections. It  
implements a Pull + Push gossip systerm. At the moment, there are two types of RPC  
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

### Core

The core of Babble is the component that maintains and computes the Hashgraph.  
The consensus algorithm, invented by Leemon Baird, is best described in the [white-paper](http://www.swirlds.com/downloads/SWIRLDS-TR-2016-01.pdf)  
and its [accompanying document](http://www.swirlds.com/downloads/SWIRLDS-TR-2016-02.pdf). 

The Hashgraph itself is a data structure that contains all the information about  
the history of the gossip and thereby grows and grows in size as gossip spreads.  
There are various strategies to keep the size of the Hashgraph limited. In our  
implementation, the **Hashgraph** object has a dependency on a **Store** object  
which contains the actual data and is abstracted behind an interface.

The current implementation of the **Store** interface uses a set of in-memory LRU  
caches which can be extended to persist stale items to disk. The size of the LRU  
caches is configurable.

### Service

The Service exposes an HTTP API to query information about a node. At the  
moment, it only exposes a **Stats** endpoint:  

```bash
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
  "undetermined_events": "24"
}
```

## Usage 

### Go
Babble is written in [Golang](https://golang.org/). Hence, the first step is to install Go which is both  
the programming language and a CLI tool for managing Go code. Go is very opinionated  
and will require you to [define a workspace](https://golang.org/doc/code.html#Workspaces) where all your go code will reside. 

### Babble and dependencies  
Clone the [repository](https://bitbucket.org/mosaicnet/babble) in the appropriate GOPATH subdirectory:

```bash
$ mkdir -p $GOPATH/src/bitbucket.org/mosaicnet/
$ cd $GOPATH/src/bitbucket.org/mosaicnet
[...]/mosaicnet$ git clone https://bitbucket.org/mosaicnet/babble.git
```
Babble uses [Glide](http://github.com/Masterminds/glide) to manage dependencies.

```bash
[...]/babble$ sudo add-apt-repository ppa:masterminds/glide && sudo apt-get update
[...]/babble$ sudo apt-get install glide
[...]/babble$ glide install
```
This will download all dependencies and put them in the **vendor** folder.

### Testing

Babble has extensive unit-testing. Use the Go tool to run tests:  
```bash
[...]/babble$ make test
```

If everything goes well, it should output something along these lines:  
```
ok      bitbucket.org/mosaicnet/babble/net      0.052s
ok      bitbucket.org/mosaicnet/babble/common   0.011s
?       bitbucket.org/mosaicnet/babble/cmd      [no test files]
?       bitbucket.org/mosaicnet/babble/cmd/dummy_client [no test files]
ok      bitbucket.org/mosaicnet/babble/hashgraph        0.174s
ok      bitbucket.org/mosaicnet/babble/node     1.699s
ok      bitbucket.org/mosaicnet/babble/proxy    0.018s
ok      bitbucket.org/mosaicnet/babble/crypto   0.028s
```

### Docker Testnet

To see Babble in action, we have provided a series of scripts to bootstrap a  
test network locally.  

Make sure you have [Docker](https://docker.com) installed.  

One of the scripts requires babble to be installed locally because it uses the  
**keygen** command to generate cryptographic key pairs in the format used by babble.  
To install babble:
```bash
[...]/babble$ make install
```

Then, run the testnet:  

```bash
[...]/babble$ cd docker
[...]/babble/docker$ make
```

Once the testnet is started, a script is automatically launched to monitor consensus  
figures:  

```
consensus_events:131055 consensus_transactions:0 events_per_second:265.53 id:0 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.24 sync_rate:1.00 transaction_pool:0 undetermined_events:26
consensus_events:131055 consensus_transactions:0 events_per_second:266.39 id:3 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.34 sync_rate:1.00 transaction_pool:0 undetermined_events:25
consensus_events:131055 consensus_transactions:0 events_per_second:267.30 id:2 last_consensus_round:14432 num_peers:3 round_events:10 rounds_per_second:29.44 sync_rate:1.00 transaction_pool:0 undetermined_events:31
consensus_events:131067 consensus_transactions:0 events_per_second:268.27 id:1 last_consensus_round:14433 num_peers:3 round_events:11 rounds_per_second:29.54 sync_rate:1.00 transaction_pool:0 undetermined_events:21
```

Running ```docker ps -a``` will show you that 8 docker containers have been launched:  
```
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

```
Indeed, each node is comprised of an App and a Babble node (cf Architecture section).

Run the **demo** script to play with the **Dummy App** which is a simple chat application
powered by the Babble consensus platform:

```
[...]/babble/docker$ make demo
```
![Demo](img/demo.png)


Finally, stop the testnet:
```
[...]/babble/docker$ make stop
```

### Terraform

We have also created a set of scripts to deploy Babble testnets in AWS. This  
requires [Terraform](https://www.terraform.io/) and authentication keys for AWS. As it would be too slow to  
copy Babble over the network onto every node, it is best to create a custom AWS image (AMI)  
with Babble preinstalled in ~/bin. Basically the Terraform scripts launch a certain  
number of nodes in a subnet and starts Babble on them.

```bash
[...]/babble$ cd terraform
[...]/babble/terraform$ make "nodes=12"
[...]/babble/terraform$ make watch # monitor Stats
[...]/babble/terraform$ make bombard # send a bunch of transactions
[...]/babble/terraform$ ssh -i babble.pem ubuntu@[public ip] # ssh into a node
[...]/babble/terraform$ make destroy #destroy resources
``` 