# BABBLE
## Consensus platform for distributed applications.  

Nodes in a distributed application require a component to communicate and order  
transactions before they get applied locally. Babble is a plug and play solution  
for this component. It uses the Hashgraph consensus algorithm which is optimal  
in terms of fault tolerance and messaging.

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
--------- SubmitTx(tx) ---- CommitTx(tx) -------
               |                |
 ==============|================|===============
 = BABBLE      |                |              =
 =             v                |              =
 =          ========================           =     
 =          = App Proxy            =           =
 =          =                      =           =
 =          ========================           =
 =                     |                       =
 =   =======================================   =
 =   = Core                                =   =
 =   =                                     =   =
 =   =  =============        ===========   =   =
 =   =  = Hashgraph =        = Store   =   =   =
 =   =  =============        ===========   =   =
 =   =                                     =   =
 =   =======================================   =
 =                     |                       =
 =   =======================================   =
 =   = Transport                           =   =
 =   =                                     =   =
 =   =======================================   =
 =                     ^                       =
 ======================|========================
                       |                  
                       v
                   
                    Network
```

### Proxy

The Babble node and the App are loosely coupled and can run in separate processes.  
They communicate via a very simple **JSON-RPC** interface over a **TCP** connection. 

The App submits transactions for consensus ordering via the **SubmitTx** endpoint  
exposed by the **App Proxy**. Babble asynchrously processes transactions and  
eventually feeds them back to the App, in consensus order,  with a **CommitTx**  
message.

Transactions are represented as raw bytes and Babble does not need to know what  
they represent. Therefore, encoding and decoding transactions is done by the App.

Apps must implement their own **Babble Proxy** to submit and receive committed  
transactions from Babble. The advantage of using a JSON-RPC API is that there is  
no restriction on the programming language for the App. It only requires a component    
that sends SubmitTx messages to Babble and exposes a TCP enpoint where Babble can  
send CommitTx messages.

When launching a Babble node, one must specify the address and port exposed by the  
Babble Proxy of the App. Is is also possible to configure which address and port  
the App Proxy exposes.

Example SubmitTx request (from App to Babble):
```json
request: {"method":"Babble.SubmitTx","params":["Y2xpZW50IDE6IGhlbGxv"],"id":0}
response: {"id":0,"result":true,"error":null}
```

Note that the API is **not** over HTTP; It is raw JSON over TCP. Here is an  
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

Babble nodes communicate whith other Babble nodes in a fully connected Peer To Peer  
network. Nodes gossip by choosing another node at random and telling them evertying  
they know about the Hashgraph that the other doesnt. The gossip protocol is  
extremely simple and serves the dual purpose of gossiping about transactions and  
about the gossip itself (the Hashgraph). The Hashraph contains enough information  
to compute a consensus ordering of transactions. 

The communication mechansim is a custom RPC protocol over TCP connections. There  
are only two types of RPC commands, **Known** and **Sync**. For example, when  
node **A** wants to sync with node **B**, it starts by sending a **Known** request  
to **B**. **B** responds with what it knows about the Hashgraph. **A** computes what  
it knows that **B** doesnt and sends the diff with a **Sync** request. Upon receiving  
the **Sync** request, **B** updates its Hashgraph accordingly and calculates the  
consensus order.

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

## Usage 

### Go
Babble is written in [Golang](https://golang.org/). Hence, the first step is to install Go which is both  
the programming language and a CLI tool for managing Go code. Go is very opinionated  
and will require you to [define a workspace](https://golang.org/doc/code.html#Workspaces) where all your gocode will reside. 

### Babble and dependencies  
Babble is a private repository on [Bitbucket](https://bitbucket.org). Get access rights from someone at  
Babble and clone the repository in the appropriate GOPATH subdirectory:

```bash
$ mkdir -p $GOPATH/src/bitbucket.org/mosaicnet/
$ cd $GOPATH/src/bitbucket.org/mosaicnet
[...]/mosaicnet$ git clone https://[username]@bitbucket.org/mosaicnet/babble.git
```
Replace **[username]** with whatever credentials you may have on Bitbucket.

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
[...]/babble$ go test $(glide novendor)
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
test network of four nodes locally.  

Make sure you have [Docker](https://docker.com) installed.  

Then, run the testnet:  

```bash
[...]/babble$ cd docker
[...]/babble/docker$ ./build-images
[...]/babble/docker$ ./build-conf
[...]/babble/docker$ ./run-testnet

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

Run the **watch** script to monitor consensus figures:
```
[...]/babble/docker$ ./watch
node1: last_consensus_round=626 consensus_events=5106 consensus_transactions=0 transaction_pool=0 undetermined_events=23 events/s=9.64 sync_rate=0.99
node2: last_consensus_round=627 consensus_events=5115 consensus_transactions=0 transaction_pool=0 undetermined_events=18 events/s=9.68 sync_rate=0.98
node3: last_consensus_round=627 consensus_events=5115 consensus_transactions=0 transaction_pool=0 undetermined_events=18 events/s=9.69 sync_rate=0.98
node4: last_consensus_round=627 consensus_events=5115 consensus_transactions=0 transaction_pool=0 undetermined_events=19 events/s=9.70 sync_rate=0.99

```

Run the **demo** scipt to play with the **Dummy App** which is a simple chat application
powered by the Babble consensus platform:

```
[...]/babble/docker$ ./demo
```
![Demo](img/demo.png)


Finally, stop the testnet:
```
[...]/babble/docker$ ./stop-testnet
```
