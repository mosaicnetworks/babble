#!/bin/bash

NODE=${1:-1}

# If we kill the client before the node is properly shutdown, it will get 
# invalid block signatures as it tries to commit blocks with empty state-hashes.
# So we send the SIGINT signal the node, give it 5 seconds to politely leave,
# and kill the client.

docker kill --signal=SIGINT node$NODE && \
sleep 5 && \
docker kill client$NODE 