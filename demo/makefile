nodes = 4
txs = 10
index = 1
fast-sync = false

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
	./scripts/run-testnet.sh $(nodes) $(fast-sync)

# monitor stats
watch: 
	./scripts/watch.sh $(nodes)

# stop testnet
stop: 
	./scripts/stop-testnet.sh

# show dummy chat application
show: 
	./scripts/show.sh client$(index)

# send transactions to multiple nodes
bombard: 
	./scripts/bombard.sh $(nodes) $(txs)

# send transactions to specific node
send:
	./scripts/send.sh $(index) $(txs)

# create new peer and join
join:
	./scripts/join.sh $(index) $(fast-sync)

# leave cleanly - removing itself from peer-set
leave:
	./scripts/leave.sh $(index)

# demo
demo:
	./scripts/demo.sh

.PHONY: up build conf start watch stop show bombard send join leave demo
	