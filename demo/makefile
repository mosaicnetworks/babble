nodes = 4
txs = 10
index = 5
target = 1

up: conf start watch

# build babble and dummy docker images for local registry. If this step is not
# run, the images will be pulled from the public registry on docker.io
build :
	cd ../docker && make demo

# generate keys and peers.json file
conf : 
	rm -rf conf
	./scripts/build-conf.sh $(nodes)

# launch testnet
start:
	./scripts/run-testnet.sh $(nodes)

# monitor stats
watch: 
	./scripts/watch.sh $(nodes)

# stop testnet
stop: 
	./scripts/stop-testnet.sh

# open terminals to use the dummy chat application
demo: 
	./scripts/demo.sh $(nodes)

# send transactions to multiple nodes
bombard: 
	./scripts/bombard.sh $(nodes) $(txs)

# send transactions to specific node
send:
	./scripts/send.sh $(target) $(txs)

# create new peer and join
join:
	./scripts/join.sh $(index)

.PHONY: up build conf start watch stop demo bombard send join
	