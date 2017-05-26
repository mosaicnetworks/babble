#!/bin/bash

set -eux
echo $0

private_ip=${1}
public_ip=${2}

ssh -q -i babble.pem -o "UserKnownHostsFile /dev/null" -o "StrictHostKeyChecking=no" \
 ubuntu@$public_ip  <<-EOF
    nohup /home/ubuntu/bin/babble run \
    --datadir=/home/ubuntu/babble_conf \
    --cache_size=10000 \
    --tcp_timeout=500 \
    --heartbeat=50 \
    --node_addr=$private_ip:1337 \
    --service_addr=0.0.0.0:8080 \
    --no_client=true > logs 2>&1 &
EOF