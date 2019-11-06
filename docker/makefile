BABBLE_VERSION=0.5.9
DUMMY_VERSION=0.5.7
WATCHER_VERSION=0.0.2
GLIDER_VERSION=0.0.4
MOBILE_VERSION=0.0.1

all: babble-image dummy-image watcher-image glider-image

demo: babble-image dummy-image watcher-image

babble-image:
	CGO_ENABLED=0 go build \
		--ldflags "-X github.com/mosaicnetworks/babble/src/version.GitCommit=`git rev-parse HEAD`" \
		-o babble/babble ../cmd/babble/
	docker build --no-cache=true -t mosaicnetworks/babble:$(BABBLE_VERSION)  -t mosaicnetworks/babble:latest babble/

dummy-image:
	CGO_ENABLED=0 go build \
		-o dummy/dummy ../cmd/dummy/
	docker build --no-cache=true -t mosaicnetworks/dummy:$(DUMMY_VERSION) -t mosaicnetworks/dummy:latest dummy/

watcher-image:
	docker build --no-cache=true -t mosaicnetworks/watcher:$(WATCHER_VERSION) -t mosaicnetworks/watcher:latest watcher/

glider-image:
	docker build --no-cache=true -t mosaicnetworks/glider:$(GLIDER_VERSION) glider/

mobile-image:
	docker build --no-cache=true -t mosaicnetworks/mobile:$(MOBILE_VERSION) mobile/



.PHONY: all demo babble-image dummy-image watcher-image glider-image mobile-image
