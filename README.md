# BABBLE

[![GoDoc](https://godoc.org/github.com/mosaicnetworks/babble?status.svg)](https://godoc.org/github.com/mosaicnetworks/babble)
[![CircleCI](https://circleci.com/gh/mosaicnetworks/babble.svg?style=svg)](https://circleci.com/gh/mosaicnetworks/babble)
[![Documentation Status](https://readthedocs.org/projects/babbleio/badge/?version=latest)](https://babbleio.readthedocs.io/en/latest/?badge=latest)
[![Go Report](https://goreportcard.com/badge/github.com/mosaicnetworks/babble)](https://goreportcard.com/report/github.com/mosaicnetworks/babble)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![Babble network](docs/assets/babble-network.png)

Babble is a distributed consensus engine designed to easily plug into any 
application. It uses peer-to-peer networking and a consensus algorithm to 
guarantee that a group of connected computers process the same commands in the 
same order.

## Table of Contents
 * [Features](#features)
 * [Consensus](#consensus)
 * [API](#api)
 * [Configuration](#configuration)
    + [Data Directory](#data-directory)
    + [Key](#key)
    + [Peers](#peers)
    + [Transport](#transport)
      - [TCP](#tcp)
      - [WebRTC](#webrtc)
    + [Store](#store)
    + [Maintenance Mode](#maintenance-mode)
    + [Service](#service)
    + [App Proxy](#app-proxy)
    + [Fast Sync](#fast-sync)
    + [Operational Parameters](#operational-parameters)
 * [Install](#install)
    + [Go](#go)
    + [Babble and Dependencies](#babble-and-dependencies)
    + [Other Requirements](#other-requirements)
    + [Testing](#testing)
    + [Build From Source](#build-from-source)
 * [Demo](#demo)
    
## Features

- **Asynchronous**: Participants have the freedom to process commands at 
                    different times.

- **Leaderless**: No participant plays a special role.

- **Byzantine Fault-Tolerant**: Supports one third of faulty nodes, including 
                                malicious behavior.

- **Finality**: Babble’s output can be used immediately, no need for block 
                confirmations.

- **Dynamic Membership**: Members can join or leave a Babble network without 
                          undermining security.

- **Fast Sync**: Joining nodes can sync straight to the current state of a 
                 network.

- **Accountability**: Auditable history of the consensus algorithm’s output.

- **Language Agnostic**: Integrate with applications written in any programming 
                         language.

- **Mobile**: Bindings for Android and iOS.

- **WebRTC**: Supports WebRTC connections for practical p2p connections.

## Consensus

We use an adaptation of the Hashgraph consensus algorithm, invented by Leemon
Baird, to which we added important features. Hashgraph is best described in the
[white-paper](http://www.swirlds.com/downloads/SWIRLDS-TR-2016-01.pdf) and its
[accompanying document](http://www.swirlds.com/downloads/SWIRLDS-TR-2016-02.pdf).
The original Hashgraph algorithm is protected by 
[patents](http://www.swirlds.com/ip/) in the USA, so anyone intending to use 
this software in the USA should take this into consideration. For a high level 
overview of the concepts behind Babble, please refer to this 
[document](http://docs.babble.io/en/latest/consensus.html).

Babble's major departure from the original Hashgraph algorithm is the 
introduction of 
[blocks](http://docs.babble.io/en/latest/blockchain.html), which represent 
self-contained sections of the Hashgraph, and which are instrumental in the 
implementation of two important new features that were alluded to in Baird's 
paper, but not specified:

- A [dynamic membership](http://docs.babble.io/en/latest/dynamic_membership.html)
  protocol, which enables peers to join or leave a group on demand.

- A [fast-sync](http://docs.babble.io/en/latest/fastsync.html) protocol which
  enables joining nodes to fast-forward straight to a point in the hashgraph
  without downloading the entire history. 

## API

![Babble design](docs/_static/babble_design_2.png)

Babble communicates with the App through an `AppProxy` interface, which has two
implementations:

- `InmemProxy` : An InmemProxy uses native callback handlers to integrate Babble
                 as a regular Go dependency.

- `SocketProxy`: A SocketProxy connects to an App via TCP sockets. It enables
                 the application to run in a separate process or machine, and to
                 be written in any programming language.

Refer to the [dummy](src/dummy) package for an example that implements both
proxies.

```go
// Start from default Babble configuration.
babbleConfig := config.NewDefaultConfig()

// Create dummy InmemProxy
dummy := NewInmemDummyClient(babbleConfig.Logger())

// Set the proxy in the Babble configuration.
babbleConfig.Proxy = dummy

// Instantiate Babble.
babble := babble.NewBabble(babbleConfig)

// Read in the configuration and initialise the node accordingly.
if err := babble.Init(); err != nil {
    babbleConfig.Logger().Error("Cannot initialize babble:", err)
    os.Exit(1)
}

// The application can submit transactions to Babble using the proxy's
// SubmitTx. Babble will broadcast the transactions to other nodes, run them
// through the consensus algorithm, and eventually call the callback methods
// implemented in the handler.
go func() {
    dummy.SubmitTx([]byte("the test transaction"))
}()

// Run the node aynchronously.
babble.Run()

// Babble reacts to SIGINT (Ctrl + c) and SIGTERM by calling the leave
// method to politely leave a Babble network, but it can also be called
// manually.
defer babble.Node.Leave()
```

## Configuration

Babble configuration is defined in the [config](src/config) package. 

### Data Directory

Babble reads configuration files from its data directory which defaults to 
`~/.babble` on Linux. It can be overwritten with `DataDir` in the Config object
or `--datadir` from the CLI.

### Key

Every Babble validator requires a cryptographic key-pair to encrypt, sign and 
verify messages. The private key is secret but the public key is used by other 
nodes to verify messages signed with the private key. The encryption scheme used
by Babble is ECDSA with the secp256k1 curve (like Bitcoin and Ethereum).

To pass a private key to Babble, either set it directly in the `Config` object, 
or dump it to a `priv_key` file in the data directory. Babble's `keygen` command
may be used to generate key-pairs in the appropriate format.

### Peers

Babble needs to know the other peers in the network. This is specified by adding
two JSON files in the data directory.

- `genesis.peers.json` corresponds to the initial validator-set; the one that
the hashgraph was started with. If `genesis.peers.json` is not provided, Babble
will use `peers.json` as the genesis validator-set.

- `peers.json` corresponds to the set of peers that the node should attempt to 
connect to upon starting.

`peers.json` and `gensesis.peers.json` are not necessarily equal because
the [dynamic membership](http://docs.babble.io/en/latest/dynamic_membership.html)
protocol enables new nodes to join or leave a live Babble network dynamically. 
It is important for a joining node to know the initial validator-set in order to 
replay and verify the hashgraph up to the point where it joins.

It is possible to start a Babble network with just a single node, or with a
predefined validator-set composed of multiple nodes. In the latter case, 
someone, or some process, needs to aggregate the public keys and network 
addresses of all participants into a single file (`peers.genesis.json`), and 
ensure that everyone has a copy of this file. It is left to the user to derive a
scheme to produce the configuration files but the docker demo 
[scripts](demo/scripts) are a good place to start.

To join an existing network, a peer must first obtain the JSON peers files from 
an existing node and place them in the data directory. One way to obtain the
peers files is to query the `/peers` and `/genesispeers` functions exposed by 
the HTTP API [service](#service). Please refer to the 
[join script](demo/scripts) in the demo for an example. 
for an example.

### Transport

Implementations of the [Transport](src/net/transport.go) interface determine 
how nodes communicate with one-another.

#### TCP

The TCP transport is suitable when nodes are in the same local network, or
when users are able to configure their connections appropriately to avoid NAT
issues.

To use a TCP transport, set the following configuration properties:

- `BinAdddr` or `--listen`: the IP:PORT of the TCP socket that Babble binds to.
  By default `BindAddr` is `127.0.0.1:1337`, meaning that Babble will bind to 
  the loopback address on the local machine.

- `AdvertiseAddr` or `--advertise`: (optional) The address that is advertised to
  other nodes. If `BindAddr` is a local address not reachable by other peers, 
  it is necessary to set `AdvertiseAddr` to something else. If `AdvertiseAddr` 
  is not set, it defaults to the `BindAddr`.

For example, when running a node from a local network behind a NAT, the 
`BindAddr` might be `192.168.1.10` which is not reachable from outside the local
network. So it is necessary to set `AdvertiseAddr` to the public IP of the 
router, and to setup port-forwarding in the NAT.

Note that the advertise address (which defaults to bind address if not set) must
match the address of the peer in the `peers.genesis.json` or `peers.json` files. 

#### WebRTC

Because Babble is a peer-to-peer application, it can run into issues with NATs 
and firewalls. The WebRTC transport addresses the NAT traversal issue, but it 
requires centralised servers for peers to exchange connection information and 
to provide STUN/TURN services. 

To use a WebRTC transport, use the following configuration properties:

- `WebRTC` or `--webrtc`: tells Babble to use a WebRTC transport.

- `SignalAddr` or `--signal-addr`: address of the WebRTC signaling server.

- `SignalRealm` or `--signal-realm`: routing domain within the signaling server.

- `ICEServers`: a slice describing servers available to be used by ICE, such as 
  STUN and TURN servers.

WebRTC requires a signaling mechanism for peers to exchange connection 
information. This requires a central server, so when the WebRTC transport is 
used, Babble is not fully p2p anymore. That being said, all the  computation and
data at the application layer remains p2p; the signaling server is only used as 
a sort of peer-discovery mechanism. We povide the code for a signaling server
[here](cmd/signal). The [demo](#demo) has a WebRTC option that illustrates the
usage of WebRTC.

It is not necessary to specify network addresses in the JSON peer files when 
WebRTC is enabled because this information will be exchanged over the signaling 
server. Likewise, the `BindAddr` and `AdvertiseAddr` options will be ignored.

The default `ICEServers` points to a public STUN server hosted by Google 
(`stun:stun.l.google.com:19302`). It does not include a TURN server, so not all 
p2p connections will be possible.

### Store

We can choose to run Babble with a database backend or only with an in-memory 
cache. With the `Store` (`--store`) option, Babble will look for a database in 
`datadir/babdger_db` or in the path specified by `DatabaseDir` (`--db`). If the 
database already exists, and the `Bootstrap` (`--boostrap`) option is set, the 
node will load the database and bootstrap itself to a state consistent with the 
database and it will be able to proceed with the consensus algorithm from there.
If the database does not exist yet, or the `Bootstrap` option is not set, a new
one will be created and the node will start from a clean state.

### Maintenance Mode

The node can also be started in `maintenance-mode` with the homonymous flag. The
node is started normally but goes straight into the `Suspended` state, where it 
still  responds to sync-requests, and service API requests, but does not produce
or insert new Events in the underlying hashgraph. The `Suspended` state is also 
triggered automatically when more than `suspend-limit`, multiplied by the number
of validators, undetermined-events were created since last starting the node. 
This is a safeguard against runaway conditions when a network does not have a 
strong  majority and produces undetermined-events ad infinitum.   

### Service

We can also specify where Babble exposes its HTTP API which provides information
about the internals of the hashgraph and data store. This is controlled by the 
`ServiceAddr` (`--service-listen`) option. It can also be disabled altogether 
with the `NoService` (`--no-service`) option.

### App Proxy

When we use Babble as a native Go library, we set the InmemProxy directly in the 
Config object's `Proxy` field. 

Instead, when Babble and the application are connected by a TCP interface, we 
start Babble as a standalone executable and we specify the endpoints of the
connection:

 - `--proxy-listen`  : where Babble listens for transactions from the App.
 - `--client-connect` : where the App listens for blocks from Babble

### Fast Sync

`EnableFastSync` (`--fast-sync`) tells Babble to attempt to fast-forward to the
tip of the hashgraph when joining, instead of downloading and replaying the 
entire hashgraph from start. More on this in 
[fast-sync](http://docs.babble.io/en/latest/fastsync.html). This options 
requires the Snapshot and Restore handlers to be carefully implemented in the
AppProxy.

### Operational Parameters

- `LogLevel` (`--log`): Determines the chattiness of the log output.

- `HeartbeatTimeout` (`--heartbeat`): Frequency of the gossip timer when there 
  is something to gossip about.

- `SlowHeartbeatTimeout` (`--slow-heartbeat`): Frequency of the gossip timer 
  when there is nothing to gossip about.

- `MaxPool` (`--max-pool`): Controls how many connections are pooled per target
  in the gossip routines.

- `TCPTimeout` (`--timeout`): Timeout of gossip RPC connections (also applies
  for WebRTC connections).

- `JoinTimeout` (`--join_timeout`): Timeout of join requests.

- `SyncLimit` (`--sync-limit`): Max number of hashgraph events to include in a
   SyncResponse or EagerSyncRequest.

- `CacheSize` (`--cache-size`): Max number of items in the in-memory caches.

- `SuspendLimit` (`--suspend-limit`): Multiplier applied to the number of 
   validators to determine the limit of undetermined events that will cause a 
   node to become suspended.

- `Moniker` (`--moniker`): Friendly name for this node. It takes precedence over
  the moniker defined in JSON peers files.

- `SignalSkipVerify` (`--signal-skip-verify`): (insecure) Controls whether the
  signal client verifies the server's certificate chain and hostname when WebRTC
  is activated.

## Install

### Go

Babble is written in [Golang](https://golang.org/) **1.14**. Hence, the first 
step is to install **Go version 1.14 or above** which is both the programming 
language and a CLI tool for managing Go code. Go is very opinionated and will 
require you to [define a workspace](https://golang.org/doc/code.html#Workspaces)
where all your go code will reside.

### Babble and Dependencies

Fetch Babble from github:

```bash
$ go get github.com/mosaicnetworks/babble
```

Download all dependencies and put them in the **vendor** folder.

```bash
[...]/babble$ make vendor
```

Babble uses `go mod` to manage dependencies.

### Other Requirements

Bash scripts used in this project assume the use of GNU versions of coreutils.
Please ensure you have GNU versions of these programs installed:-

example for macOS:

```bash
# --with-default-names makes the `sed` and `awk` commands default to gnu sed and gnu awk respectively.
brew install gnu-sed gawk --with-default-names
```

### Testing

Babble has extensive unit-testing.

```bash
[...]/babble$ make test
```

If everything goes well, it should output something along these lines:

```text
?       github.com/mosaicnetworks/babble/src/babble     [no test files]
ok      github.com/mosaicnetworks/babble/src/common     0.015s
ok      github.com/mosaicnetworks/babble/src/crypto     0.122s
ok      github.com/mosaicnetworks/babble/src/hashgraph  10.270s
?       github.com/mosaicnetworks/babble/src/mobile     [no test files]
ok      github.com/mosaicnetworks/babble/src/net        0.012s
ok      github.com/mosaicnetworks/babble/src/node       19.171s
ok      github.com/mosaicnetworks/babble/src/peers      0.038s
?       github.com/mosaicnetworks/babble/src/proxy      [no test files]
ok      github.com/mosaicnetworks/babble/src/dummy        0.013s
ok      github.com/mosaicnetworks/babble/src/proxy/inmem        0.037s
ok      github.com/mosaicnetworks/babble/src/proxy/socket       0.009s
?       github.com/mosaicnetworks/babble/src/proxy/socket/app   [no test files]
?       github.com/mosaicnetworks/babble/src/proxy/socket/babble        [no test files]
?       github.com/mosaicnetworks/babble/src/service    [no test files]
?       github.com/mosaicnetworks/babble/src/version    [no test files]
?       github.com/mosaicnetworks/babble/cmd/babble     [no test files]
?       github.com/mosaicnetworks/babble/cmd/babble/commands    [no test files]
?       github.com/mosaicnetworks/babble/cmd/dummy      [no test files]
?       github.com/mosaicnetworks/babble/cmd/dummy/commands     [no test files]

```

### Build From Source

The easiest way to build binaries is to do so in a hermetic Docker container.
Use this simple command:

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

## Demo

To see Babble in action, we have provided a series of scripts to bootstrap a
local test network with the dummy application and the SocketProxy.

**NOTE:**
This has been tested on Ubuntu 18.04 and macOS.

Make sure you have [Docker](https://docker.com) installed.

Then, run the testnet:

```bash
[...]/babble$ cd demo
[...]/babble/demo$ make
# or using webrtc
[...]/babble/demo$ make webrtc=true
```

Once the testnet is started, a script is automatically launched to monitor each
Babble node:

```text
consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:1 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:18
consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:3 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:20
consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:2 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:21
consensus_events:180 consensus_transactions:40 events_per_second:0.00 id:0 last_block_index:3 last_consensus_round:17 num_peers:3 round_events:7 rounds_per_second:0.00 state:Babbling sync_rate:1.00 transaction_pool:0 undetermined_events:20
```

Running `docker ps -a` will show you that 9 docker containers have been launched:

```bash
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
```

Indeed, each replica is composed of a dummy application coupled to a Babble node
running in a different container.

Run the **demo** script to play with the **Dummy App** which is a simple chat
application powered by the Babble consensus platform:

```bash
[...]/babble/demo$ make demo
```

![Demo](demo/img/demo.png)

Finally, stop the testnet:

```bash
[...]/babble/demo$ make stop
```