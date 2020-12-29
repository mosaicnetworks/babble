BABBLE_VERSION=0.8.4dev
DUMMY_VERSION=0.8.0dev
SIGNAL_VERSION=0.0.2
WATCHER_VERSION=0.0.2
GLIDER_VERSION=0.0.5
MOBILE_VERSION=0.0.2

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

signal-image:
	CGO_ENABLED=0 go build \
		-o signal/signal ../cmd/signal/
	docker build --no-cache=true -t mosaicnetworks/signal:$(SIGNAL_VERSION) -t mosaicnetworks/signal:latest signal/

watcher-image:
	docker build --no-cache=true -t mosaicnetworks/watcher:$(WATCHER_VERSION) -t mosaicnetworks/watcher:latest watcher/

glider-image:
	docker build --no-cache=true -t mosaicnetworks/glider:$(GLIDER_VERSION) glider/

mobile-image:
	docker build --no-cache=true -t mosaicnetworks/mobile:$(MOBILE_VERSION) mobile/



.PHONY: all demo babble-image dummy-image signal-image watcher-image glider-image mobile-image
