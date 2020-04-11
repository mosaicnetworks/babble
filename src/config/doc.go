// Package config defines the configuration for a Babble node.
//
// Regardless of how Babble is started, directly from Go code or as a standalone
// process from the command line, it uses the Config object defined in this
// package to store and forward configuration options. On top of these
// configuration options, Babble relies on a data directory, defined by
// Config.DataDir, where it expects to find a few additional configuration
// files:
//
//  priv_key // a plain text file containing the raw private key (cf. babble keygen).
//  peers.json // a JSON file containing the current list of peers.
//  peers.genesis.json // (optional, defaults to peers.json) a JSON file containing the initial list of peers.
//  cert.pem // (optional) an x509 certificate for the WebRTC signaling server.
package config
