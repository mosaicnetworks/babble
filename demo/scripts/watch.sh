#!/bin/bash

# docker run -it --rm --name=watcher --net=babblenet --ip=172.77.5.99 mosaicnetworks/watcher /watch.sh $N  

watch -t -n 1 '
docker ps -aqf name=node | \
xargs docker inspect -f "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}" | \
xargs -I % curl -s -m 1 http://%:80/stats | \
tr -d "{}\"" | \
awk -F "," '"'"'{gsub (/[,]/," "); print;}'"'"'
'