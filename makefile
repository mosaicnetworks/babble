
install: 
	go install github.com/babbleio/babble/cmd/babble
test: 
	glide novendor | xargs go test

.PHONY: install test
	