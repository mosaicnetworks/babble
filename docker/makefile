all: babble-image dummy-image

babble-image:
	CGO_ENABLED=0 go build \
		--ldflags "-X github.com/babbleio/babble/version.GitCommit=`git rev-parse HEAD`" \
		-o babble/babble ../cmd/babble/
	docker build --no-cache=true -t babble babble/

dummy-image:
	CGO_ENABLED=0 go build \
		--ldflags "-X github.com/babbleio/babble/version.GitCommit=`git rev-parse HEAD`" \
		-o dummy/dummy ../cmd/dummy_client/
	docker build --no-cache=true -t dummy dummy/

.PHONY: all babble-image dummy-image