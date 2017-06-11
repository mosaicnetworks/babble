#!/bin/bash

docker ps -f name=client -f name=node -q | xargs docker rm -f 
docker network rm babblenet