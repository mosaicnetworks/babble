
nodes = 4
txs = 1000

up: conf create start

conf : 
	rm -rf conf
	./scripts/build-conf.sh $(nodes)

create :
	rm -f ips.dat
	terraform apply -var-file=secret.tfvars -var "servers=$(nodes)"	

start:
ifdef index
	awk 'FNR==$(index) {system("./scripts/remote-run.sh "$$0" "$$1"")}' ips.dat
else
	awk '{system("./scripts/remote-run.sh "$$0" "$$1"")}' ips.dat  
endif

watch : 
	./scripts/watch.sh

bombard: 
	./scripts/bombard.sh $(txs)

stop: 
ifdef index
	awk 'FNR==$(index) {system("./scripts/remote-kill.sh "$$0" "$$1"")}' ips.dat 
else
	awk '{system("./scripts/remote-kill.sh "$$0" "$$1"")}' ips.dat
endif

destroy :
	terraform destroy -var-file=secret.tfvars
	
.PHONY: up conf create start watch bombard stop destroy
	