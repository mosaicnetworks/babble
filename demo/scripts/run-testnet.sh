#!/bin/bash

set -eux

N=${1:-4}
MPWD=$(pwd)

docker network create \
  --driver=bridge \
  --subnet=172.77.0.0/16 \
  --ip-range=172.77.0.0/16 \
  --gateway=172.77.5.254 \
  babblenet

for i in $(seq 1 $N)
do
    docker run -d --name=client$i --net=babblenet --ip=172.77.10.$i -it mosaicnetworks/dummy:0.4.0 \
    --name="client $i" \
    --client-listen="172.77.10.$i:1339" \
    --proxy-connect="172.77.5.$i:1338" \
    --discard \
    --log="debug" 
done

for i in $(seq 1 $N)
do
    docker create --name=node$i --net=babblenet --ip=172.77.5.$i mosaicnetworks/babble:0.4.0 run \
    --heartbeat=100ms \
    --moniker="node$i" \
    --cache-size=50000 \
    --listen="172.77.5.$i:1337" \
    --proxy-listen="172.77.5.$i:1338" \
    --client-connect="172.77.10.$i:1339" \
    --service-listen="172.77.5.$i:80" \
    --sync-limit=500 \
    --log="debug" 
    #--store \
    
    docker cp $MPWD/conf/node$i node$i:/.babble
    docker start node$i
done
