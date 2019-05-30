# Peer Sets Synch etc

## Changes Made

Added a new configuration parameter --enable-fast-sync that sets Node.Config.EnableFastSync. Default value is true, so that there are no changes to the current behaviour, so hopefully it will not break everything.

## To Do

The check as to whether to Fast Sync is in Node.push, using the node servicing the request's parameters, not the requesting node.

This should be changed by amending the args in node_rpc.requestSync as called in pull, to make them available in push.

Idle or Paused Babble Node state, with a mechanism to close/reopen the tcp sockets.

//XXX comments are things that need improving.
