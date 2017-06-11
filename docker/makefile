nodes = 4

up: build conf start watch

build :
	./scripts/build-images.sh

conf : 
	rm -rf conf
	./scripts/build-conf.sh $(nodes)

start:
	./scripts/run-testnet.sh $(nodes)

watch: 
	./scripts/watch.sh $(nodes)

stop: 
	./scripts/stop-testnet.sh

demo: 
	./scripts/demo.sh $(nodes)

.PHONY: up build conf start watch stop demo
	