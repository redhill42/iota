# This file describes the standard way to build Iota, using Docker
#
# Usage:
#
# # Assemble the full dev environment. This is slow the first time.
# docker build -t iota-dev .
#
# # Mount your source in an interactive container for quick testing:
# docker run -v `pwd`:/go/src/github.com/redhill42/iota --privileged -i -t iota-dev bash
#
# # Run the test suite:
# docker run --privileged iota-dev build/make.sh test

FROM icloudway/dev:latest

ENV GOPATH /go

ARG PROXY
RUN http_proxy=${PROXY} https_proxy=${PROXY} go get -u github.com/onsi/ginkgo/ginkgo github.com/onsi/gomega

# Compile Go for cross compilation
ENV CROSSPLATFORMS \
    linux/amd64 linux/arm \
    windows/amd64
#    darwin/amd64 #FIXME

WORKDIR /go/src/github.com/redhill42/iota

VOLUME /data

# Wrap all commands in the "docker-in-docker" script to allow nested containers
ENTRYPOINT ["build/dind"]

# Upload source
COPY . /go/src/github.com/redhill42/iota
