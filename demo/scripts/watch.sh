#!/bin/bash

N=${1:-4}

#docker run -it --rm --name=watcher --net=babblenet --ip=172.77.5.99 mosaicnetworks/watcher /watch.sh $N  

watch -t -n 1 '
for i in $(seq 1 '$N');
do
    curl -s -m 1 http://172.77.5.$i:80/stats | \
        tr -d "{}\"" | \
        awk -F "," '"'"'{gsub (/[,]/," "); print;}'"'"'
done;
'