FROM circleci/golang:1.9.2-stretch
# Install Glide (Go dependency manager)
RUN curl https://glide.sh/get | sh
# Install Gox (Go cross-compiler)
RUN go get -u -v github.com/mitchellh/gox
# Install ghr (Creates github releases and uploads artifacts)
RUN go get -u -v github.com/tcnksm/ghr