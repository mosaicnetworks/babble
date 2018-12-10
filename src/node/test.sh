#!/bin/bash

for i in `seq 1 100`
do
    go test -run TestJoinFull  > ~/gossip.logs
    if grep "FAIL" ~/gossip.logs 
    then
        echo 'CHECK LOGS'
        exit
    else
        echo $i "OK"
    fi
done
echo 'NO ERRORS'
