import machine
from machine import Timer
from umqtt.simple import MQTTClient
import dht
import network
import ubinascii
import time
import sys
import json

WIFI_SSID = 'YOUR_SSID'
WIFI_PASS = 'YOUR_PASSWORD'
MQTT_HOST = '192.168.55.1'
MQTT_PORT = 1883
MQTT_TOKEN = "YOUR_ACCESS_TOKEN"
MQTT_CLIENT_ID = ubinascii.hexlify(machine.unique_id())

def connect_to_wifi():
    wlan = network.WLAN(network.STA_IF)
    wlan.active(True)
    if not wlan.isconnected():
        print('connecting to network...')
        wlan.connect(WIFI_SSID, WIFI_PASS)
        while not wlan.isconnected():
            pass
    print('network config:', wlan.ifconfig())

def connect_to_mqtt():
    client = MQTTClient(MQTT_CLIENT_ID, MQTT_HOST, MQTT_PORT, MQTT_TOKEN, "")
    client.connect()
    print('Connected to MQTT broker')
    return client

def restart_and_reconnect():
    print('Failed to connect to MQTT broker. Reconnecting...')
    time.sleep(10)
    machine.reset()

d = dht.DHT11(machine.Pin(15))
def measure_temp(c):
    topic = "api/v1/%s/me/measurement" % MQTT_TOKEN
    try:
        d.measure()
        c.publish(topic, "dht,measure=temp value=%d" % (d.temperature()))
        c.publish(topic, "dht,measure=humi value=%d" % (d.humidity()))
    except:
        pass

try:
    connect_to_wifi()
    client = connect_to_mqtt()
    tmr = Timer(-1)
    tmr.init(period=2000, mode=Timer.PERIODIC, callback=lambda t:measure_temp(client))
    while True:
        client.wait_msg()
except:
    restart_and_reconnect()
