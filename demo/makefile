nodes = 4
txs = 10

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

# send a bunch of transactions
bombard: 
	./scripts/bombard.sh $(nodes) $(txs)

# create new peer and join
join:
	./scripts/join.sh $(nodes)

.PHONY: up build conf start watch stop demo bombard join
	