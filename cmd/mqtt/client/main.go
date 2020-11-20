package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/redhill42/iota/cmd/mqtt/common"
)

func main() {
	token := flag.String("t", "", "The device token")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "No device token provided")
		return
	}

	client, err := common.DialMQTT()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to MQTT broker: %v\n", err)
		return
	}
	defer client.Disconnect(250)

	requestTopic := *token + "/me/rpc/request/1"
	responseTopic := *token + "/me/rpc/response/1"
	choke := make(chan string)

	handler := func(client mqtt.Client, msg mqtt.Message) {
		choke <- string(msg.Payload())
	}
	if t := client.Subscribe(responseTopic, 0, handler); t.Wait() && t.Error() != nil {
		fmt.Fprintf(os.Stderr, "Failed to subscribe: %v\n", t.Error())
		return
	}

	req := `{"id":1, "method":"sayHello", "param":{"message":"hello, world"}}`
	if t := client.Publish(requestTopic, 0, false, req); t.Wait() && t.Error() != nil {
		fmt.Fprintf(os.Stderr, "Failed to publish: %v\n", t.Error())
	}

	timer := time.NewTimer(time.Second * 5)
	select {
	case res := <-choke:
		fmt.Println(res)
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		fmt.Fprintln(os.Stderr, "RPC time out")
	}
}
