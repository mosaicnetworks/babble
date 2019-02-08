#!/bin/bash

set -u

NODE=${1:-1}
COUNT=${2:-1}

for i in `seq 1 $COUNT`; do
    printf "Node$NODE Tx$i" | base64 | \
    xargs printf "{\"method\":\"Babble.SubmitTx\",\"params\":[\"%s\"],\"id\":$i}" | \
    nc -v -w 1 -N 172.77.5.$NODE 1338
done; 