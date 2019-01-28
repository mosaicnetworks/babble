#!/bin/bash

set -u

NODES=${1:-4}
COUNT=${2:-10}

for i in `seq 1 $COUNT`; do
	for n in `seq 1 $NODES`; do
        printf "Node$n Tx$i" | base64 | \
        xargs printf "{\"method\":\"Babble.SubmitTx\",\"params\":[\"%s\"],\"id\":$i}" | \
        nc -v -w 1 -N 172.77.5.$n 1338
    done;
done; 

