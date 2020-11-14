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

# Compile Go for cross compilation
ENV IOTA_CROSSPLATFORMS \
    linux/386 linux/arm \
    darwin/amd64 \
    windows/amd64 windows/386

WORKDIR /go/src/github.com/redhill42/iota

VOLUME /data

# Wrap all commands in the "docker-in-docker" script to allow nested containers
ENTRYPOINT ["build/dind"]

# Upload source
COPY . /go/src/github.com/redhill42/iota
