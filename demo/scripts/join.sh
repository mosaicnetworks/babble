#!/bin/bash

N=${1:-5}
DEST=${2:-"$PWD/conf"}

dest=$DEST/node$N

# Create new key-pair and place it in new conf directory
mkdir -p $dest
echo "Generating key pair for node$N"
docker run  \
    -v $dest:/.babble \
    --rm mosaicnetworks/babble:0.4.0 keygen 

# get up-to-date peers.json
echo "Fetching peers.json from node1"
curl -s http://172.77.5.1:80/peers > $dest/peers.json
cat $dest/peers.json 

# start the new node
docker run -d --name=client$N --net=babblenet --ip=172.77.10.$N -it mosaicnetworks/dummy:0.4.0 \
    --name="client $N" \
    --client-listen="172.77.10.$N:1339" \
    --proxy-connect="172.77.5.$N:1338" \
    --discard \
    --log="debug" 

docker create --name=node$N --net=babblenet --ip=172.77.5.$N mosaicnetworks/babble:0.4.0 run \
    --cache-size=50000 \
    --listen="172.77.5.$N:1337" \
    --proxy-listen="172.77.5.$N:1338" \
    --client-connect="172.77.10.$N:1339" \
    --service-listen="172.77.5.$N:80" \
    --sync-limit=1000 \
    --store \
    --log="debug"
    #--store \

docker cp $dest node$N:/.babble
docker start node$N

