# BABBLE
## Consensus platform for distributed applications.  

[![CircleCI](https://circleci.com/gh/babbleio/babble.svg?style=svg)](https://circleci.com/gh/babbleio/babble)

Babble allows many computers to behave as one. It uses Peer to Peer (P2P) networking  
and a consensus algorithm to guarantee that multiple connected computers process  
the same commands in the same order; a technique known as state machine replication.  
This makes for secure systems that can tolerate arbitrary failures including malicious  
behaviour.

For guidance on how to install and use Babble please visit our [documentation](http://docs.babble.io) pages.

**NOTE**:  
This is alpha software. Please contact us if you intend to run it in production.

## Consensus Algorithm

We use the Hashgraph consensus algorithm, invented by Leemon Baird.  
It is best described in the [white-paper](http://www.swirlds.com/downloads/SWIRLDS-TR-2016-01.pdf) and its [accompanying document](http://www.swirlds.com/downloads/SWIRLDS-TR-2016-02.pdf).   
The algorithm is protected by [patents](http://www.swirlds.com/ip/) in the USA. Therefore, anyone intending to  
use this software in the USA should obtain a license from the patent holders.

Hashgraph is based on the intuitive idea that gossiping about gossip itself yields  
enough information to compute a consensus ordering of events. It attains the  
theoretical limit of tolerating up to one-third of faulty nodes without compromising  
on speed. For those familiar with the jargon, it is a leaderless, asynchronous BFT  
consensus algorithm.

## Design

Babble is designed to integrate with applications written in any programming language.  


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

## Build from source

The easiest way to build binaries is to do so in a hermetic Docker container. Use  
this simple command:  

```bash
[...]/babble$ make dist
```
This will launch the build in a Docker container and write all the artifacts in  
the build/ folder.  

```bash
[...]/babble$ tree build
build/
├── dist
│   ├── babble_0.1.0_darwin_386.zip
│   ├── babble_0.1.0_darwin_amd64.zip
│   ├── babble_0.1.0_freebsd_386.zip
│   ├── babble_0.1.0_freebsd_arm.zip
│   ├── babble_0.1.0_linux_386.zip
│   ├── babble_0.1.0_linux_amd64.zip
│   ├── babble_0.1.0_linux_arm.zip
│   ├── babble_0.1.0_SHA256SUMS
│   ├── babble_0.1.0_windows_386.zip
│   └── babble_0.1.0_windows_amd64.zip
└── pkg
    ├── darwin_386
    │   └── babble
    ├── darwin_amd64
    │   └── babble
    ├── freebsd_386
    │   └── babble
    ├── freebsd_arm
    │   └── babble
    ├── linux_386
    │   └── babble
    ├── linux_amd64
    │   └── babble
    ├── linux_arm
    │   └── babble
    ├── windows_386
    │   └── babble.exe
    └── windows_amd64
        └── babble.exe
```

## Dev

### Go
Babble is written in [Golang](https://golang.org/). Hence, the first step is to install **Go version 1.9 or above**  
which is both the programming language  and a CLI tool for managing Go code. Go is  
very opinionated and will require you to [define a workspace](https://golang.org/doc/code.html#Workspaces) where all your go code will  
reside. 

### Babble and dependencies  
Clone the [repository](https://github.com/babbleio/babble) in the appropriate GOPATH subdirectory:

```bash
$ mkdir -p $GOPATH/src/github.com/babbleio/
$ cd $GOPATH/src/github.com/babbleio
[...]/babbleio$ git clone https://github.com/babbleio/babble.git
```
Babble uses [Glide](http://github.com/Masterminds/glide) to manage dependencies.

```bash
[...]/babble$ curl https://glide.sh/get | sh
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
ok      github.com/babbleio/babble/net      0.052s
ok      github.com/babbleio/babble/common   0.011s
?       github.com/babbleio/babble/cmd      [no test files]
?       github.com/babbleio/babble/cmd/dummy_client [no test files]
ok      github.com/babbleio/babble/hashgraph        0.174s
ok      github.com/babbleio/babble/node     1.699s
ok      github.com/babbleio/babble/proxy    0.018s
ok      github.com/babbleio/babble/crypto   0.028s
```

## Demo

To see Babble in action, we have provided a series of scripts to bootstrap a test  
network locally.  

**NOTE:**  
This has only been tested on Ubuntu 16.04

Make sure you have [Docker](https://docker.com) installed.  

Then, run the testnet:  

```bash
[...]/babble$ cd docker
[...]/babble/demo$ make
```

Once the testnet is started, a script is automatically launched to monitor consensus  
figures:  

```
consensus_events:98 consensus_transactions:40 events_per_second:0.00 id:3 last_consensus_round:11 num_peers:3 round_events:12 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:34
consensus_events:98 consensus_transactions:40 events_per_second:0.00 id:1 last_consensus_round:11 num_peers:3 round_events:12 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:35
consensus_events:98 consensus_transactions:40 events_per_second:0.00 id:0 last_consensus_round:11 num_peers:3 round_events:12 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:34
consensus_events:98 consensus_transactions:40 events_per_second:0.00 id:2 last_consensus_round:11 num_peers:3 round_events:12 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:35

```

Running ```docker ps -a``` will show you that 8 docker containers have been launched:  
```
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

```
Indeed, each node is comprised of an App and a Babble node (cf Architecture section).

Run the **demo** script to play with the **Dummy App** which is a simple chat application
powered by the Babble consensus platform:

```
[...]/babble/demo$ make demo
```
![Demo](img/demo.png)


Finally, stop the testnet:
```
[...]/babble/demo$ make stop
```

