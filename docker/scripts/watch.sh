#!/bin/bash


N=${1:-4}

watch -n 1 '
for i in $(seq 1 '$N');
do
    curl -s http://172.77.5.$i:80/Stats | \
    tr -d "{}\"" | \
    awk -F "," '"'"'{gsub (/[,]/," "); print;}'"'"'
done;
'    
