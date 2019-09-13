# Change Log

## Unreleased

SECURITY:
FEATURES:
IMPROVEMENTS:
BUG FIXES:

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
