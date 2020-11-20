package mqtt

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
	"github.com/redhill42/iota/config"
	"github.com/sirupsen/logrus"
)

type Broker struct {
	client mqtt.Client
	Mux    *mux.Router
	qos    byte
}

func NewBroker() (*Broker, error) {
	broker := new(Broker)
	opts := broker.configure()

	broker.client = mqtt.NewClient(opts)
	if token := broker.client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	// Subscribe mqtt topic and forward to API server. The topic has the
	// following pattern
	//
	//    api / <ver> / <token> / <verb> / #
	//            ^        ^        ^      ^-- extra components in request URI
	//            |        |        |-- HTTP methods: GET, POST, DELETE
	//            |        |-- device access token
	//            |-- API version number, v1, v2, etc
	//
	// for example, the MQTT topic "api/v1/XXX/post/me/attributes" will
	// forwarded to API server with URL "/api/v1/me/attributes".

	if token := broker.client.Subscribe("api/+/+/+/#", broker.qos, broker.serveMQTT); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return broker, nil
}

func (broker *Broker) configure() *mqtt.ClientOptions {
	server := config.GetOption("mqtt", "url")
	user := config.GetOption("mqtt", "user")
	password := config.GetOption("mqtt", "password")
	clientId := config.GetOption("mqtt", "clientId")
	qos := config.GetOption("mqtt", "qos")

	if server == "" {
		server = "tcp://127.0.0.1:1883"
	}

	if clientId == "" {
		buf := make([]byte, 16)
		rand.Read(buf)
		clientId = hex.EncodeToString(buf)
	}

	if qos != "" {
		qosValue, err := strconv.Atoi(qos)
		if err != nil || qosValue < 0 || qosValue > 2 {
			logrus.Warnf("mqtt: Invalid Quality of Service level: %s", qos)
		} else {
			broker.qos = byte(qosValue)
		}
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetClientID(clientId)
	opts.SetUsername(user)
	opts.SetPassword(password)

	return opts
}

func (broker *Broker) Close() {
	broker.client.Disconnect(250)
}

type fakeWriter struct {
	header http.Header
}

func (w *fakeWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *fakeWriter) Write(p []byte) (int, error) {
	// discard output
	return len(p), nil
}

func (w *fakeWriter) WriteHeader(statusCode int) {
	// don't actually write
}

func (broker *Broker) serveMQTT(client mqtt.Client, msg mqtt.Message) {
	sp := strings.SplitN(msg.Topic(), "/", 5)
	if len(sp) != 5 {
		logrus.Errorf("Invalid topic: %s", msg.Topic())
		return
	}
	version, token, method, path := sp[1], sp[2], strings.ToUpper(sp[3]), sp[4]

	uri := "/api/" + version + "/" + path
	body := bytes.NewReader(msg.Payload())
	r, err := http.NewRequest(method, uri, body)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create request")
		return
	}

	r.Header.Set("Authorization", "bearer "+token)
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
	}

	w := fakeWriter{}
	broker.Mux.ServeHTTP(&w, r)
}
