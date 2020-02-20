FROM circleci/golang:1.13.0
# It is necessary to overwrite the GOCACHE variable
ENV GOCACHE /tmp/.cache
# Install Gox (Go cross-compiler)
RUN go get -u -v github.com/mitchellh/gox