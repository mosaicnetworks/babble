#!/bin/bash

N=${1:-4}

docker run -it --rm --name=watcher --net=babblenet --ip=172.77.5.99 mosaicnetworks/watcher /watch.sh $N  