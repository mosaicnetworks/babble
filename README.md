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

### Creating a Babble Network

The files contained in the docker demo presented above should contain helpfull  
inforation to understand how Babble networks can be configured and implemented.

Basically, the steps are as follows:  
1) Create a network of machines where the nodes will be deployed. Note the addresses  
and ports where each node will be listening for Babble messages.  

2) Create a private-public key pair for each node using the **babble keygen** utility.  
   In production scenarios, you are more likely to ask each participant to generate   
   their key pair and send you only the public key. Private keys are stored in  
   individual **priv_key.pem** files and should remain secret.  

3) Create the **peers.json** file which contains the list of all participants.  

4) Copy the **peers.json** and **priv_key.pem** files in the appropritate folder  
   on each node.  

5) Install Babble and App on nodes  

6) Run Babble with appropriate flags   


**Babble Keygen** is a utility that generates cryptographic key pairs in the  
format used by Babble.

Example:
```
$ babble keygen
PublicKey:
0x044CA00BEE8CBF2ABA4086D9C407F23223552951878DC63DEC3B8E9D127B29A4B79B4037563B690CEDABFE3A3A86DE9255577A0ABFF3357619E8789DD54A9A9F0B
PrivateKey:
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIPCUYVgk3o5dTmT97MMgpHQKxK4AKb6KNwpjfe3eZdRVoAoGCCqGSM49
AwEHoUQDQgAETKAL7oy/KrpAhtnEB/IyI1UpUYeNxj3sO46dEnsppLebQDdWO2kM
7av+OjqG3pJVV3oKv/M1dhnoeJ3VSpqfCw==
-----END EC PRIVATE KEY-----
```

One would typically extract the Private Key part and copy it into a **priv_key.pem**  
file.

The **peers.json** file list all the participants in the network and identifies  
them by public key and address.
  
Example peers.json file defining four peers:
```
[
    {
        "NetAddr":"172.77.5.1:1337",
        "PubKeyHex":"0x0445E0DA94DFC18BF38C8F0C95C1B21A2AE248CCD1CD490C83CDE9FF9FD43DEA7F59E294010386FDFE71F3FEDBCD6507803F49758733C349279610D54C3D82DD9B"
    },
    {
        "NetAddr":"172.77.5.2:1337",
        "PubKeyHex":"0x04AF55528C5648AE08E3B7BB1F2FB0B6F758145DA3B489D3FBA122B92317E50D31E75834AA2F405421326C2F4B24868AB3578E09D26F6A4F243B631FA06A5645FA"
    },
    {
        "NetAddr":"172.77.5.3:1337",
        "PubKeyHex":"0x044D9F2AD5A17CB1D15CACAACCA4306F565F8957A48E0D246330DEBC47B3385FF3F085D67AF02295400DE96062489B49420965BED0C2181309DEB4290FD307C01D"
    },
    {
        "NetAddr":"172.77.5.4:1337",
        "PubKeyHex":"0x0475010E2F079A8012FBA55C7BEDB963950325822E6837B6AF21D92B3239C7E6C321BF3632BA365FEE2CA35DFC4B045A02DB10A501EEE4091EB8470B9012B44937"
    }
]
```
Each Babble node requires a **peers.json** and a **priv_key.pem** file. **peers.json**  
files must be identical but the **priv_key.pem** is unique to each node and must correspond  
to one of the public keys listed in the **peers.json** file.

Both files must reside in the folder specified to Babble by the **datadir** flag  
which default to ~/.babble on linux machines.

```bash
---:~/.babble$ tree
.
├── peers.json
└── priv_key.pem

```

Run a node

```
NAME:
   babble run - Run node

USAGE:
   babble run [command options] [arguments...]

OPTIONS:
   --datadir value      Directory for the configuration (default: "/home/<user>/.babble")
   --node_addr value    IP:Port to bind Babble (default: "127.0.0.1:1337")
   --no_client          Run Babble with dummy in-memory App client
   --proxy_addr value   IP:Port to bind Proxy Server (default: "127.0.0.1:1338")
   --client_addr value  IP:Port of Client App (default: "127.0.0.1:1339")
   --log_level value    debug, info, warn, error, fatal, panic (default: "debug")
   --heartbeat value    Heartbeat timer milliseconds (time between gossips) (default: 1000)
   --max_pool value     Max number of pooled connections (default: 2)
   --tcp_timeout value  TCP timeout milliseconds (default: 1000)
   --cache_size value   Number of items in LRU caches (default: 500)

```

Example:  
```
$ babble run --node_addr="172.77.5.1:1337" --proxy_addr="172.77.5.1:1338" --client_addr="172.77.5.1:1339" 
```

Here Babble listens to other Babble nodes on 172.77.5.1:1337.  
It listens to the App on 172.77.5.1:1338  
It expects the App to be listening on 172.77.5.1.1339