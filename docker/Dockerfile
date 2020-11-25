FROM debian:buster as builder

WORKDIR /app

RUN apt-get clean && apt-get update && apt-get install -y --no-install-recommends \
    build-essential libssl-dev tar wget \
    && rm -rf /var/lib/apt/lists/*

# Build mosquitto
ENV MOSQUITTO_VERSION=1.6.12
RUN wget http://mosquitto.org/files/source/mosquitto-${MOSQUITTO_VERSION}.tar.gz \
    && tar xzvf mosquitto-${MOSQUITTO_VERSION}.tar.gz \
    && rm mosquitto-${MOSQUITTO_VERSION}.tar.gz \
    && cd mosquitto-${MOSQUITTO_VERSION} \
    && make && make install \
    && groupadd -r mosquitto && useradd -r -g mosquitto mosquitto

# Start from a new image
FROM debian:buster

WORKDIR /app

RUN apt-get clean && apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates gnupg tar wget \
    && rm -rf /var/lib/apt/lists/*

# Setup mosquitto env.
RUN mkdir -p /var/lib/mosquitto /var/log/mosquitto \
    && groupadd -r mosquitto \
    && useradd -r -g mosquitto mosquitto \
    && chown -R mosquitto:mosquitto /var/lib/mosquitto \
    && chown -R mosquitto:mosquitto /var/log/mosquitto

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
    && mkdir -p /data/db \
    && chown -R mongodb:mongodb /data/db \
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

# Install influxdb
ENV INFLUXDB_VERSION 2.0.2
RUN set -x \
    && groupadd -r influxdb && useradd -r -g influxdb influxdb \
    && mkdir -p /data/influxdb /var/log/influxd \
    && chown -R influxdb:influxdb /data/influxdb \
    && wget -O - https://dl.influxdata.com/influxdb/releases/influxdb-${INFLUXDB_VERSION}_linux_amd64.tar.gz | tar -xz \
    && mkdir -p /app/bin/ \
    && mv influxdb-${INFLUXDB_VERSION}_linux_amd64/influx* /app/bin/ \
    && rm -rf influxdb-${INFLUXDB_VERSION}_linux_amd64

COPY . /app
COPY --from=builder /usr/local/sbin/mosquitto /app/bin/
ENV IOTA_CONFIG_FILE /app/conf/iota.conf
ENV INFLUX_CONFIGS_PATH /app/conf/influx.conf
VOLUME /data
EXPOSE 8080 1883 8086
ENTRYPOINT ["./docker-entry.sh"]
