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

FROM debian:buster

RUN mkdir -p /var/cache/apt/archives && touch /var/cache/apt/archives/lock
RUN apt-get clean && apt-get update && apt-get install -y --no-install-recommends \
    apt-utils \
    automake \
    bash-completion \
    binutils-mingw-w64 \
    bsdmainutils \
    build-essential \
    ca-certificates \
    clang-3.8 \
    createrepo \
    curl \
    dpkg-sig \
    gcc-mingw-w64 \
    git \
    gnupg \
    ssh-client \
    jq \
    libtool \
    mercurial \
    net-tools \
    pkg-config \
    python-dev \
    python-mock \
    python-pip \
    python-websocket \
    redis-server \
    tar \
    wget \
    zip \
    && ln -snf /usr/bin/clang-3.8 /usr/local/bin/clang \
    && ln -snf /usr/bin/clang++-3.8 /usr/local/bin/clang++ \
    && rm -rf /var/lib/apt/lists/*

# Configure the container for OSX cross compilation
ENV OSX_SDK MacOSX10.11.sdk
ENV OSX_CROSS_COMMIT 8aa9b71a394905e6c5f4b59e2b97b87a004658a4
RUN set -x \
    && export OSXCROSS_PATH="/osxcross" \
    && git clone https://github.com/tpoechtrager/osxcross.git $OSXCROSS_PATH \
    && (cd $OSXCROSS_PATH && git checkout -q $OSX_CROSS_COMMIT) \
    && curl -sSL https://s3.dockerproject.org/darwin/v2/${OSX_SDK}.tar.xz -o "${OSXCROSS_PATH}/tarballs/${OSX_SDK}.tar.xz" \
    && UNATTENDED=yes OSX_VERSION_MIN=10.6 ${OSXCROSS_PATH}/build.sh \
    && (cd /osxcross/target/SDK/MacOSX10.11.sdk/usr/include/c++ && ln -s 4.2.1 v1)
ENV PATH /osxcross/target/bin:$PATH

# Install Go
# IMPORTANT: If the version of Go is updated, the Windows to Linux CI machines
#            will need updating, to avoid errors.
ENV GO_VERSION 1.15.5
RUN curl -fsSL "https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -xzC /usr/local
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

#############################
# Setup testing environment #
#############################

# grab gosu for easy step-down from root
ENV GOSU_VERSION 1.7
RUN set -x \
    && wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$(dpkg --print-architecture)" \
    && wget -O /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$(dpkg --print-architecture).asc" \
    && export GNUPGHOME="$(mktemp -d)" \
    && gpg --keyserver ha.pool.sks-keyservers.net --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4 \
    && gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu \
    && rm -r "$GNUPGHOME" /usr/local/bin/gosu.asc \
    && chmod +x /usr/local/bin/gosu \
    && gosu nobody true

# Install mongo db
ENV MONGO_MAJOR 4.4
ENV MONGO_VERSION 4.4.2
RUN set -x \
    && groupadd -r mongodb && useradd -r -g mongodb mongodb \
    && wget -qO - https://www.mongodb.org/static/pgp/server-$MONGO_MAJOR.asc | apt-key add - \
    && echo "deb http://repo.mongodb.org/apt/debian buster/mongodb-org/$MONGO_MAJOR main" > /etc/apt/sources.list.d/mongodb-org.list \
    && export DEBIAN_FRONTEND=noninteractive \
    && ln -s /bin/true /bin/systemctl \
	&& apt-get update && apt-get install -y \
		mongodb-org=$MONGO_VERSION \
		mongodb-org-server=$MONGO_VERSION \
		mongodb-org-shell=$MONGO_VERSION \
		mongodb-org-mongos=$MONGO_VERSION \
		mongodb-org-tools=$MONGO_VERSION \
    && rm -f /bin/systemctl \
	&& rm -rf /var/lib/apt/lists/* \
	&& rm -rf /var/lib/mongodb \
	&& mv /etc/mongod.conf /etc/mongod.conf.orig

# Compile Go for cross compilation
ENV CROSSPLATFORMS \
    linux/amd64 linux/arm \
    darwin/amd64 \
    windows/amd64

WORKDIR /project/iota

VOLUME /data

# Wrap all commands in the "docker-in-docker" script to allow nested containers
ENTRYPOINT ["build/dind"]

# Upload source
COPY . /project/iota
