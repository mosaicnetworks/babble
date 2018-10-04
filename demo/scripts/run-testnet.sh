#!/bin/bash

set -eux

N=${1:-4}
MPWD=$(pwd)

docker network create \
  --driver=bridge \
  --subnet=172.77.0.0/16 \
  --ip-range=172.77.5.0/24 \
  --gateway=172.77.5.254 \
  babblenet

for i in $(seq 1 $N)
do
    docker run -d --name=client$i --net=babblenet --ip=172.77.5.$(($N+$i)) -it mosaicnetworks/dummy:0.3.0 \
    --name="client $i" \
    --client_addr="172.77.5.$(($N+$i)):1339" \
    --proxy_addr="172.77.5.$i:1338" \
    --log_level="info" 
done

for i in $(seq 1 $N)
do
    docker create --name=node$i --net=babblenet --ip=172.77.5.$i mosaicnetworks/babble:0.3.0 run \
    --cache-size=50000 \
    --timeout=200ms \
    --heartbeat=10ms \
    --listen="172.77.5.$i:1337" \
    --proxy-listen="172.77.5.$i:1338" \
    --client-connect="172.77.5.$(($N+$i)):1339" \
    --service-listen="172.77.5.$i:80" \
    --sync-limit=1000 \
    --store = true \
    --log="debug"

    docker cp $MPWD/conf/node$i node$i:/.babble
    docker start node$i
done
