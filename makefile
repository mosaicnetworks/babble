
install: 
	go install bitbucket.org/mosaicnet/babble/cmd/babble
test: 
	glide novendor | xargs go test

.PHONY: install test
	