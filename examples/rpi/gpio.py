import RPi.GPIO as GPIO
import paho.mqtt.client as mqtt
import json
import logging
from jsonrpc import JSONRPCResponseManager, dispatcher

IOTA_HOST = "192.168.55.1"
TOKEN = "YOUR_ACCESS_TOKEN"

class GPIOStatus:
    def __init__(self, pin, publisher):
        self.pin = pin
        self.status = False
        self.publisher = publisher

        GPIO.setmode(GPIO.BCM)
        GPIO.setup(pin, GPIO.OUT)
        GPIO.output(pin, GPIO.LOW)

    def get_status(self):
        return self.status

    def set_status(self, status):
        self.status = status
        GPIO.output(self.pin, GPIO.HIGH if status else GPIO.LOW)
        self.upload()
        return status

    def toggle(self):
        return self.set_status(not self.status)

    def upload(self):
        self.publisher.publish('api/v1/'+TOKEN+'/me/attributes', json.dumps({"status": self.status}))


class Device:
    def __init__(self):
        client = mqtt.Client()
        client.on_connect = self.on_connect
        client.on_message = self.on_message
        client.username_pw_set(TOKEN)

        gpio = GPIOStatus(17, client)
        dispatcher.add_method(gpio.get_status)
        dispatcher.add_method(gpio.set_status)
        dispatcher.add_method(gpio.toggle)

        self.client = client
        self.gpio = gpio
        client.connect(IOTA_HOST, 1883, 60)

    # The callback for when the client receives a CONNACK response from the server
    def on_connect(self, client, userdata, flags, rc):
        print("Connected with result code "+str(rc))
        # Subscribing to receive RPC requests
        client.subscribe(TOKEN+"/me/rpc/request/+")
        # Sending current GPIO status
        self.gpio.upload()

    # The callback for when a PUBLISH message is received from the server
    def on_message(self, client, userdata, msg):
        print('Message: ' + str(msg.payload))
        response = JSONRPCResponseManager.handle(msg.payload, dispatcher)
        if response is not None :
            client.publish(msg.topic.replace('request', 'response'), response.json)

    def loop(self):
        try:
            self.client.loop_forever()
        except KeyboardInterrupt:
            GPIO.cleanup()


def main():
    device = Device()
    device.loop()


if __name__ == '__main__':
    logging.basicConfig()
    main()
