FROM circleci/golang:1.13.0
# It is necessary to overwrite the XDG_CACHE_HOME
ENV  XDG_CACHE_HOME=/tmp/.cache
# Install Gox (Go cross-compiler)
RUN go get -u -v github.com/mitchellh/gox