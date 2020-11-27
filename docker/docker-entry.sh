#!/bin/bash
set -e

gosu mongodb mongod --fork --logpath /var/log/mongodb/mongod.log --bind_ip 127.0.0.1 >/dev/null
gosu mosquitto /app/bin/mosquitto -d -c /app/conf/mosquitto.conf

# start influxdb
gosu influxdb /app/bin/influxd \
    --engine-path=/data/influxdb/engine \
    --bolt-path=/data/influxdb/influxd.bolt \
    2>&1 >/var/log/influxd/influxd.log &

# wait for influxdb started
timeout 30 bash -c 'until printf "" 2>>/dev/null >>/dev/tcp/$0/$1; do echo -n "."; sleep 1; done' localhost 8086

# one time setup for influxdb
if ! [ -e $INFLUX_CONFIGS_PATH ]; then
    # generate a random password
    password=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32};echo)
    /app/bin/influx setup -u iota -p $password -o iota -b iota -f
    # extract token from influxdb config file
    token=$(grep -oP '^  token = "\K[^"]+' $INFLUX_CONFIGS_PATH)
    # add password and token to iota configuration
    /app/bin/iota config influxdb.password $password
    /app/bin/iota config influxdb.token $token
fi

exec /app/bin/iota apiserver
