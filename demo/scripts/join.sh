#!/bin/bash

N=${1:-4}
DEST=${2:-"$PWD/conf"}

n=$((N+1))

# Create new key-pair and place it in new conf directory
dest=$DEST/node$n
mkdir -p $dest
echo "Generating key pair for node$n"
docker run  \
    -v $dest:/.babble \
    --rm mosaicnetworks/babble:0.4.0 keygen 

# copyt the existing peers.json file
cp $DEST/peers.json $dest/

# start the new node

docker run -d --name=client$n --net=babblenet --ip=172.77.5.$(($N+$n)) -it mosaicnetworks/dummy:0.4.0 \
    --name="client $n" \
    --client-listen="172.77.5.$(($N+$n)):1339" \
    --proxy-connect="172.77.5.88:1338" \
    --discard \
    --log="debug" 

docker create --name=node$n --net=babblenet --ip=172.77.5.88 mosaicnetworks/babble:0.4.0 run \
    --cache-size=50000 \
    --listen="172.77.5.88:1337" \
    --proxy-listen="172.77.5.88:1338" \
    --client-connect="172.77.5.$(($N+$n)):1339" \
    --service-listen="172.77.5.88:80" \
    --sync-limit=1000 \
    --log="debug"
    # --store \
    
docker cp $DEST/node$n node$n:/.babble
docker start node$n

