#!/bin/bash

set -eux

private_ip=${1}
public_ip=${2}

ssh -q -i babble.pem -o "UserKnownHostsFile /dev/null" -o "StrictHostKeyChecking=no" \
 ubuntu@$public_ip "killall -9 babble dummy"