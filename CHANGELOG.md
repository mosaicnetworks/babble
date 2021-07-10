# Change Log

## v0.8.5 (Jul 10, 2021)

IMPROVEMENTS:

- proxy: implement retry mechanism for socket proxies

## v0.8.4 (Dec 29, 2020)

BUG FIXES:

- hashgraph: stopping condition in updateAncestorFirstDescendant

## v0.8.3 (Oct 10, 2020)

IMPROVEMENTS:

- hashgraph: BFT block timestamp

## v0.8.2 (Oct 3, 2020)

FEATURES:

- hashgraph: block timestamp

IMPROVEMENTS:

- node: Remove unused stats.

## v0.8.1 (June 3, 2020)

IMPROVEMTS:

- config: Expose ICE server configuration.

BUG FIXES:

- net: Add timeouts in Signaling mechanism.

## v0.8.0 (April 28, 2020)

FEATURES:

- transport: WebRTC transport

IMPROVEMENTS:

- doc: Improve comments and package documentation
- dependencies: Switch from using glide to go mod 

BUG FIXES:

- transport: Do not initialise transport when maintenance-mode is activated

## v0.7.0 (February 20, 2020)

FEATURES:

- proxy: BREAKING API CHANGE - New OnStateChanged method in AppProxy

IMPROVEMENTS:

- babble: Dynamic SuspendLimit. SuspendLimit is now multiplied to the number of
          validators.

- dependencies: Switch to using go mod for dependency management, instead of 
                glide. 

BUG FIXES:

- mobile: Use fork of badgerdb in mobile. Fixes the directory lock issue with
          Android 6 (Issue #20 in babble-android).

## v0.6.2 (February 7, 2020)

IMPROVEMENTS:

- mobile: Update build scripts and Dockerimage

## v0.6.1 (January 30, 2020)

IMPROVEMENTS:

- mobile: BREAKING CHANGE - Use file-based configuration

## v0.6.0 (December 20, 2019)

IMPROVEMENTS:

- mobile: BREAKING CHANGE - OnCommit handler uses blocks.
- mobile: Disable API service.
- mobile: Use low slow-heartbeat. 
- node: Configurable slow-heartbeat.

BUG FIXES:

- hashgraph: Change stopping-condition in updateAncestorsFirstDescendants.
- node: Change eviction-condition in checkSuspend.

## v0.5.12 (November 27, 2019)

IMPROVEMENTS:

- node: Suspend node when the validator is evicted.
- net: Enable joining with hostname, not just numeric IP. 
- cmd: Remove Standalone mode.

BUG FIXES:

- keys: Remove 600 permissions check on priv_key, which makes Babble work in
        Windows.

## v0.5.11 (November 14, 2019)

FEATURES:

- node: Automatically suspend a node when the number of undetermined events 
        exceeds the limit specified by the suspend-limit flag 300 by default).

BUG FIXES:

- store: Disable database writes during bootstrap process. This bug was causing
         the Badger DB to double in size each time the node was boostrapped.

## v0.5.10 (November 7, 2019)

IMPROVEMENTS:

- spike: Documented a potential solution to automatically evict faulty peers.
- node: Add a suspended state which allows running a node without opening the 
        network transport and starting the gossip routines.

BUG FIXES:

- core: Include Events from unknown participants in EventDiff
- node: Fix issue #128 regarding Extra tests
- store: Fix cache issues (issue #127) 

## v0.5.9 (October 25, 2019)

BUG FIXES:

- badger_db: Fix error reading events expired from the cache
- badger_db: Add the truncate option to truncate value log files to delete
             corrupt data
- node:      Defer network starting listening after bootstrap process ends


## v0.5.8 (October 15, 2019)

IMPROVEMENTS:

- badger_db: Update to use latest v1.6.0 release 
- config: Enable setting a different database directory with --db


## v0.5.7 (October 2, 2019)

IMPROVEMENTS:

- node: Add timestamp to stats 

BUG FIXES:

- node: Intercept SIGTERM as well as SIGINT to leave politely.

## v0.5.6 (September 18, 2019)

FEATURES:

- net: Add configuration to advertise a different address than "BindAddr" to 
       the rest of the cluster.

## v0.5.5 (September 13, 2019)

IMPROVEMENTS:

- hashgraph: better handling of "normal" SelfParent errors

## v0.5.4 (September 6, 2019)

FEATURES:

- service: Add endpoint `/blocks/{startIndex}?limit={x}`
- service: Deprecated `/block/{x}` endpoint
- service: Add endpoint `/validators/{round}`
- service: Add endpoint `/history`

IMPROVEMENTS:

* node: Limit number of concurrent gossip routines

## v0.5.3 (August 29, 2019)

FEATURES:

- service: Enable CORS

## v0.5.2 (August 8, 2019)

IMPROVEMENTS:

- log: Custom formatter for logrus. More info logs.

BUG FIXES:

- mobile: Process InternalTransactions

## v0.5.1 (July 23, 2019)

BUG FIXES:

- core: Do nothing on 'leave' when the node is alone.

## v0.5.0 (July 14, 2019)

SECURITY:

- crypto: Use secp256k1 curve which is also used by Ethereum and Bitcoin.

FEATURES:

- hashgraph: Dynamic Membership protocol.

IMPROVEMENTS:

- node: FastSync protocol.
- core: Bootstrap.
- Enhanced peers leaving and joining functionality
- Added peers endpoint to enable `peers.json` propagation

BUG FIXES:

## v0.4.1 (January 28, 2019)

IMPROVEMENTS:

- Bug fixes and refactoring pulled from develop branch

## v0.4.0 (October 14, 2018)

SECURITY:

- keygen: write keys to files instead of tty.

FEATURES:

- proxy: Introduced in-memory proxy.
- cmd: Enable reading config from file (babble.toml, .json, or .yaml)

IMPROVEMENTS:

- node: major refactoring of configuration and initialization of Babble node.
- node: Node ID is calculated from public key rather than from sorting the
  peers.json file.

## v0.3.0 (September 4, 2018)

FEATURES:

- hashgraph: Replaced Leemon Baird's original "Fair" ordering method with
  Lamport timestamps.
- hashgraph: Introduced the concept of Frames and Roots to enable initializing a
  hashgraph from a "non-zero" state.
