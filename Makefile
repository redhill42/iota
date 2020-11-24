.PHONY: all binary build cross default shell test validate help

PROJECT=iota
HOST_OSARCH := $(shell go env GOOS)/$(shell go env GOARCH)

# get OS/Arch of docker engine
DOCKER_OSARCH := $(shell bash -c 'source build/make/.detect-daemon-osarch && echo $${DOCKER_ENGINE_OSARCH:-$$DOCKER_CLIENT_OSARCH}')
DOCKERFILE := $(shell bash -c 'source build/make/.detect-daemon-osarch && echo $${DOCKERFILE}')

# env vars passed through directly to Docker's build scripts
# to allow things link `make KEEPBUNDLE=1 binary` easily
DOCKER_ENVS := \
    -e BUILDFLAGS \
    -e KEEPBUNDLE \
    -e PROJ_DEBUG \
    -e PROJ_GITCOMMIT \
    -e INCREMENTAL_BINARY \
    -e TESTDIRS \
    -e TESTFLAGS \
    -e DOCKER_REGISTRY_MIRROR \
    -e CROSS \
    -e COVER \
    -e TIMEOUT \
    -e http_proxy -e https_proxy

# to allow `make BIND_DIR=. shell` or `make BIND_DIR= test`
# (default to no bind mount if DOCKER_HOST is set)
BIND_DIR := $(if $(BINDDIR),$(BINDDIR),bundles)
DOCKER_MOUNT := $(if $(BIND_DIR),-v "$(CURDIR)/$(BIND_DIR):/project/$(PROJECT)/$(BIND_DIR)")
MODULE_MOUNT := -v "$(CURDIR)/modules:/go/pkg/mod"

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_BRANCH_CLEAN := $(shell echo $(GIT_BRANCH) | sed -e "s/[^[:alnum:]]/-/g")
DOCKER_IMAGE := iota-dev$(if $(GIT_BRANCH_CLEAN),:$(GIT_BRANCH_CLEAN))

DOCKER_FLAGS := docker run --rm -i --privileged $(DOCKER_ENVS) $(DOCKER_MOUNT) $(MODULE_MOUNT) -v docker-data:/var/lib/docker

# if this session isn't interactive, then we don't want to allocate a
# TTY, which would fail, but if it is interactive, we do want to attach
# so that the user can send e.g. ^C through.
INTERACTIVE := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(INTERACTIVE), 1)
	DOCKER_FLAGS += -t
endif

DOCKER_RUN_DOCKER := $(DOCKER_FLAGS) "$(DOCKER_IMAGE)"

default: build gofmt
	CROSS=$(HOST_OSARCH) $(DOCKER_RUN_DOCKER) build/make.sh binary

all: build ## validate all checks, cross build the binaries, \nrun all test and generate archives
	$(DOCKER_RUN_DOCKER) build/make.sh

build: bundles
	docker build ${DOCKER_BUILD_ARGS} -t "$(DOCKER_IMAGE)" -f "$(DOCKERFILE)" .

gofmt:
	@build/gofmt.sh

bundles:
	mkdir bundles

binary: build ## cross build the binaries for linux, darwin and windows
	$(DOCKER_RUN_DOCKER) build/make.sh binary

tgz: build ## build the archive containing the binaries
	$(DOCKER_RUN_DOCKER) build/make.sh binary tgz

shell: build ## start a shell inside the build env
	$(DOCKER_RUN_DOCKER) bash

test: build ## run the tests
	CROSS=linux/amd64 $(DOCKER_RUN_DOCKER) build/make.sh binary test-unit cover

validate: build ## validate gofmt, go vet
	$(DOCKER_RUN_DOCKER) build/make.sh validate-gofmt

clean:
	@docker images | grep '<none>' | awk '{print $3}' | xargs docker rmi 2>/dev/null || true

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%21c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
