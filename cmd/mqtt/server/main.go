package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
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

	topic := *token + "/me/rpc/request/+"
	if t := client.Subscribe(topic, 0, serveRPC); t.Wait() && t.Error() != nil {
		fmt.Fprintf(os.Stderr, "Subscribe failed: %v\n", t.Error())
		return
	}

	time.Sleep(time.Hour)
}

type RPCRequest struct {
	Id     int
	Method string
	Param  struct {
		Message string
	}
}

func serveRPC(client mqtt.Client, msg mqtt.Message) {
	sp := strings.Split(msg.Topic(), "/")
	sp[len(sp)-2] = "response"
	responseTopic := strings.Join(sp, "/")

	var req RPCRequest
	err := json.Unmarshal(msg.Payload(), &req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unmarshal failed: %v", err)
	}

	fmt.Printf("%s\n", req.Param.Message)
	client.Publish(responseTopic, 0, false, `{"result":"ok"}`)
}
