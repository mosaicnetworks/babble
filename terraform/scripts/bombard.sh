#!/bin/bash

set -u

COUNT=${1:-1000}

privs=($(cat ips.dat | awk '{ print $1 }'))
pubs=($(cat ips.dat | awk '{ print $2 }'))

for i in `seq 1 $COUNT`; do
   for n in "${!privs[@]}"; do 
        printf "${privs[$n]} Tx$i" | base64 | \
        xargs printf "{\"method\":\"Babble.SubmitTx\",\"params\":[\"%s\"],\"id\":$i}" | \
        nc -v -w 1 ${pubs[$n]} 1338
    done;
done; 

