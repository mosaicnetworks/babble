#!/bin/bash

N=${1:-4}

watch -t -n 1 '
for i in $(seq 1 '$N');
do
    curl -s -m 1 http://172.77.5.$i:80/stats | \
        tr -d "{}\"" | \
        awk -F "," '"'"'{gsub (/[,]/," "); print;}'"'"'
done;
'