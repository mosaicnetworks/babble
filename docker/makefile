BABBLE_VERSION=0.1.1
DUMMY_VERSION=0.1.0
GLIDER_VERSION=0.0.2

all: babble-image dummy-image glider-image 

demo: babble-image dummy-image

babble-image:
	CGO_ENABLED=0 go build \
		--ldflags "-X github.com/babbleio/babble/version.GitCommit=`git rev-parse HEAD`" \
		-o babble/babble ../cmd/babble/
	docker build --no-cache=true -t mosaicnetworks/babble:$(BABBLE_VERSION) babble/

dummy-image:
	CGO_ENABLED=0 go build \
		-o dummy/dummy ../cmd/dummy_client/
	docker build --no-cache=true -t mosaicnetworks/dummy:$(DUMMY_VERSION) dummy/

glider-image:
	docker build --no-cache=true -t mosaicnetworks/glider:$(GLIDER_VERSION) glider/

.PHONY: all demo babble-image dummy-image glider-image