package mqtt

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/redhill42/iota/config"
	"github.com/sirupsen/logrus"
)

type Logger logrus.Level

func (l Logger) Println(v ...interface{}) {
	logrus.StandardLogger().Logln(logrus.Level(l), v...)
}

func (l Logger) Printf(format string, v ...interface{}) {
	logrus.StandardLogger().Logf(logrus.Level(l), format, v...)
}

func init() {
	mqtt.ERROR = Logger(logrus.ErrorLevel)
	mqtt.CRITICAL = Logger(logrus.FatalLevel)
}

type Broker struct {
	client mqtt.Client
	mux    http.Handler
	qos    byte
	tokenQ chan mqtt.Token
}

func NewBroker(username, password string) (*Broker, error) {
	broker := new(Broker)
	opts := broker.configure(username, password)
	broker.client = mqtt.NewClient(opts)

	broker.tokenQ = make(chan mqtt.Token, 100)
	go broker.drainTokenQ()
	return broker, nil
}

func (broker *Broker) configure(username, password string) *mqtt.ClientOptions {
	server := config.GetOrDefault("mqtt.url", "tcp://127.0.0.1:1883")

	qosStr := config.GetOrDefault("mqtt.qos", "1")
	qos, err := strconv.Atoi(qosStr)
	if err != nil || qos < 0 || qos > 2 {
		logrus.Warnf("mqtt: Invalid Quality of Service level: %s", qosStr)
		qos = 1
	}
	broker.qos = byte(qos)

	clean, _ := strconv.ParseBool(config.GetOrDefault("mqtt.clean", "false"))

	clientid := config.GetOrDefault("mqtt.clientid", "")
	if clientid == "" {
		buf := make([]byte, 16)
		rand.Read(buf)
		clientid = hex.EncodeToString(buf)
		config.AddOption("mqtt", "clientid", clientid)
		config.Save()
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetClientID(clientid)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetCleanSession(clean)

	opts.SetDefaultPublishHandler(func(_ mqtt.Client, msg mqtt.Message) {
		go broker.serveMQTT(msg)
	})

	return opts
}

func (broker *Broker) drainTokenQ() {
	for {
		t, more := <-broker.tokenQ
		if !more || t == nil {
			break
		}
		_ = t.Wait()
		if t.Error() != nil {
			logrus.WithError(t.Error()).Error("Failed to publish message")
		}
	}
}

func (broker *Broker) Publish(topic string, payload interface{}) (err error) {
	switch payload.(type) {
	case string, []byte, bytes.Buffer:
		// message type is ok
	default:
		// must encode to json
		if payload, err = json.Marshal(payload); err != nil {
			return err
		}
	}
	broker.tokenQ <- broker.client.Publish(topic, broker.qos, false, payload)
	return nil
}

func (broker *Broker) Subscribe(topic string, callback func(string, []byte)) error {
	t := broker.client.Subscribe(topic, broker.qos, func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), msg.Payload())
	})
	t.Wait()
	return t.Error()
}

func (broker *Broker) Unsubscribe(topic string) {
	broker.tokenQ <- broker.client.Unsubscribe(topic)
}

func (broker *Broker) Close() {
	broker.tokenQ <- nil
	broker.client.Disconnect(250)
}

const apiTopic = "api/#"

// Subscribe mqtt topic and forward to API server. The topic has the
// following pattern:
//
//    api / <ver> / <token> / <path> [ /request/$request_id ]
//            ^        ^        ^
//            |        |        |-- path of request uri
//            |        |-- device access token
//            |-- API version number, v1, v2, etc
//
// for example, the MQTT topic "api/v1/XXX/me/attributes" will
// forwarded to API server with URL "/api/v1/me/attributes".
//
// If the path ends with "/request/$request_id", where $request_id is a request
// identifier allocated by client, then this is a GET request. Client must subscribe
// to a response topic to receive response message. The response topic has the form
//    <token>/<path>/response/$request_id
// Note that the response topic doesn't contains "api/v1" prefix.
//
// for example, to get device attributes, device send an empty message to
// "api/v1/XXX/me/attributes/request/1" and subscribe to "XXX/me/attributes/response/1"
// to receive the result.
func (broker *Broker) Forward(mux http.Handler) error {
	if broker.mux != nil {
		panic("MQTT broker already forwarded")
	}
	broker.mux = mux

	// Connect to MQTT broker
	t := broker.client.Connect().(*mqtt.ConnectToken)
	if t.Wait() && t.Error() != nil {
		return t.Error()
	}

	// Subscribe on api topic if no session present
	if !t.SessionPresent() {
		logrus.Debugf("Subscribe to %s", apiTopic)
		t := broker.client.Subscribe(apiTopic, broker.qos, nil)
		if t.Wait() && t.Error() != nil {
			return t.Error()
		}
	}

	return nil
}

type fakeWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (w *fakeWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *fakeWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *fakeWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (broker *Broker) serveMQTT(msg mqtt.Message) {
	logrus.Debugf("received message: %s, %s\n", msg.Topic(), string(msg.Payload()))
	if !strings.HasPrefix(msg.Topic(), "api/") {
		return // not our message
	}
	sp := strings.Split(msg.Topic(), "/")
	if len(sp) < 4 {
		logrus.Errorf("Invalid topic: %s", msg.Topic())
		return
	}

	var version, token, method, path, requestId string

	// Parse request topic
	if len(sp) == 4 && sp[2] == "me" && sp[3] == "claim" {
		// special case for api/v1/me/claim, there is no token in the topic
		version, method, path = sp[1], "POST", "me/claim"
	} else {
		version, token = sp[1], sp[2]
		if len(sp) >= 6 && sp[len(sp)-2] == "request" {
			method = "GET"
			requestId = sp[len(sp)-1]
			path = strings.Join(sp[3:len(sp)-2], "/")
		} else {
			method = "POST"
			path = strings.Join(sp[3:], "/")
		}
	}

	var r *http.Request
	var err error

	// Create fake HTTP request
	apiPath := "/api/" + version + "/" + path
	if method == "GET" {
		if len(msg.Payload()) == 0 {
			r, err = http.NewRequest(method, apiPath, nil)
		} else {
			var q map[string]string
			err := json.Unmarshal(msg.Payload(), &q)
			if err != nil {
				logrus.WithError(err).Errorf("Invalid query parameter: %s", string(msg.Payload()))
				return
			}

			query := make(url.Values)
			for k, v := range q {
				query.Add(k, v)
			}
			u := url.URL{
				Path:     apiPath,
				RawQuery: query.Encode(),
			}
			r, err = http.NewRequest(method, u.String(), nil)
		}
	} else {
		body := bytes.NewReader(msg.Payload())
		r, err = http.NewRequest(method, apiPath, body)
	}
	if err != nil {
		logrus.WithError(err).Error("Failed to create request")
		return
	}

	if token != "" {
		r.Header.Set("Authorization", "bearer "+token)
	}
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
	}

	// Route to API server
	w := fakeWriter{}
	broker.mux.ServeHTTP(&w, r)

	// Send response message
	if method == "GET" {
		responseTopic := token + "/" + path + "/response/" + requestId
		if w.statusCode < 200 || w.statusCode >= 300 {
			_ = broker.Publish(responseTopic, map[string]interface{}{
				"$status": w.statusCode,
				"$error":  string(w.body.Bytes()),
			})
		} else {
			_ = broker.Publish(responseTopic, w.body.Bytes())
		}
	}
}
