#!/bin/bash
set -e

mosq_pid=0
influx_pid=0
iota_pid=0

function start() {
    # start mongodb
    gosu mongodb mongod --fork --logpath /var/log/mongodb/mongod.log --bind_ip 127.0.0.1 >/dev/null

    # start mosquitto
    gosu mosquitto /app/bin/mosquitto -c /app/conf/mosquitto.conf & mosq_pid=$!

    # start influxdb
    gosu influxdb /app/bin/influxd \
        --engine-path=/data/influxdb/engine \
        --bolt-path=/data/influxdb/influxd.bolt \
        >/var/log/influxd/influxd.log 2>&1 & influx_pid=$!

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

    # start iota server
    /app/bin/iota apiserver & iota_pid=$!
}

function stop() {
    echo "Exiting..."
    gosu mongodb mongod --shutdown >/dev/null

    if [ $mosq_pid -ne 0 ]; then
        kill -SIGTERM "$mosq_pid"
        wait "$mosq_pid"
    fi
    if [ $influx_pid -ne 0 ]; then
        kill -SIGTERM "$influx_pid"
        wait "$influx_pid"
    fi
    if [ $iota_pid -ne 0 ]; then
        kill -SIGTERM "$iota_pid"
        wait "$iota_pid"
    fi
    exit 0
}

# setup handlers
# on callback, kill the last background process, which is
# `tail -f /dev/null` and execute the specified handler
trap 'kill ${!}; stop' SIGTERM

# run application
start "$@"

# wait forever
while true; do
    tail -f /dev/null & wait ${!}
done
