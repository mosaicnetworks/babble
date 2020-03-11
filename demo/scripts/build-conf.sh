#!/bin/bash

# This script creates the configuration for a Babble testnet with a variable
# number of nodes. It will generate crytographic key pairs and assemble a 
# peers.json file in the format used by Babble. The files are copied into 
# individual folders for each node so that these folders can be used as the 
# datadir that Babble reads configuration from.

set -e

N=${1:-4}
WEBRTC=${2:-false}
DEST=${3:-"$PWD/conf"}
IPBASE=${4:-172.77.5.}
PORT=${5:-1337}

for i in $(seq 1 $N)
do
    dest=$DEST/node$i
    mkdir -p $dest
    echo "Generating key pair for node$i"
    docker run  \
        -u $(id -u) \
        -v $dest:/.babble \
        --rm mosaicnetworks/babble:latest keygen 
    echo "$IPBASE$i:$PORT" > $dest/addr
done

PFILE=$DEST/peers.json
echo "[" > $PFILE
for i in $(seq 1 $N)
do
    com=","
    if [[ $i == $N ]]; then
        com=""
    fi

    printf "\t{\n" >> $PFILE
    if "$WEBRTC"; then
        printf "\t\t\"NetAddr\":\"$(cat $DEST/node$i/key.pub)\",\n" >> $PFILE
    else
        printf "\t\t\"NetAddr\":\"$(cat $DEST/node$i/addr)\",\n" >> $PFILE
    fi
    printf "\t\t\"PubKeyHex\":\"$(cat $DEST/node$i/key.pub)\",\n" >> $PFILE
    printf "\t\t\"Moniker\":\"node$i\"\n" >> $PFILE
    printf "\t}%s\n"  $com >> $PFILE

done
echo "]" >> $PFILE

for i in $(seq 1 $N)
do
    dest=$DEST/node$i
    cp $DEST/peers.json $dest/
    cp $DEST/peers.json $dest/peers.genesis.json
    cp $PWD/../src/net/signal/wamp/test_data/cert.pem $dest/cert.pem
done



