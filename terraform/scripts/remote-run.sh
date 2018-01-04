#!/bin/bash

set -eux
echo $0

private_ip=${1}
public_ip=${2}

ssh -q -i babble.pem -o "UserKnownHostsFile /dev/null" -o "StrictHostKeyChecking=no" \
 ubuntu@$public_ip  <<-EOF
    nohup /home/ubuntu/bin/babble run \
    --datadir=/home/ubuntu/babble_conf \
    --store=inmem \
    --cache_size=10000 \
    --tcp_timeout=500 \
    --heartbeat=50 \
    --node_addr=:1337 \
    --proxy_addr=:1338 \
    --client_addr=:1339 \
    --service_addr=:8080 > babble_logs 2>&1 &

    nohup /home/ubuntu/bin/dummy \
    --name=$private_ip \
    --client_addr=:1339 \
    --proxy_addr=:1338 < /dev/null > dummy_logs 2>&1 &
EOF