#!/bin/bash

CLIENT=${1:-"client1"}

docker exec $CLIENT tail -f dummy_debug.log