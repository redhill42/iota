package common

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/redhill42/iota/config"
)

func DialMQTT() (mqtt.Client, error) {
	config.InitializeClient()

	server := config.GetOption("mqtt", "url")
	user := config.GetOption("mqtt", "user")
	password := config.GetOption("mqtt", "password")

	if server == "" {
		server = "tcp://127.0.0.1:1883"
	}

	buf := make([]byte, 16)
	rand.Read(buf)
	clientId := hex.EncodeToString(buf)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetClientID(clientId)
	opts.SetUsername(user)
	opts.SetPassword(password)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	} else {
		return client, nil
	}
}
