#!/bin/bash

docker ps -f name=client -f name=node -f name=watcher -aq | xargs docker rm -f 
docker network rm babblenet