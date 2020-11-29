package agent

import (
	"github.com/redhill42/iota/auth"
	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/device"
	"github.com/redhill42/iota/mqtt"
	"github.com/redhill42/iota/tsdb"

	// Load all plugins
	_ "github.com/redhill42/iota/auth/userdb/file"
	_ "github.com/redhill42/iota/auth/userdb/mongodb"
)

// Agent maintains all external services
type Agent struct {
	Users         *userdb.UserDatabase
	Authz         *auth.Authenticator
	DeviceManager *device.Manager
	MQTTBroker    *mqtt.Broker
	TSDB          tsdb.TSDB
}

func New() (agent *Agent, err error) {
	agent = new(Agent)

	agent.Users, err = userdb.Open()
	if err != nil {
		return nil, err
	}

	agent.Authz, err = auth.NewAuthenticator(agent.Users)
	if err != nil {
		return nil, err
	}

	var username = config.GetOption("mqtt", "user")
	var password = config.GetOption("mqtt", "password")
	if username == "" && password == "" {
		pw, err := agent.Users.GetPassword("mqtt", 32)
		if err != nil {
			return nil, err
		}
		username = "iota"
		password = string(pw)
	}
	agent.MQTTBroker, err = mqtt.NewBroker(username, password)
	if err != nil {
		return nil, err
	}

	agent.TSDB, err = tsdb.New()
	if err != nil {
		return nil, err
	}

	agent.DeviceManager, err = device.NewManager(agent.MQTTBroker)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// Close shutdown all external services
func (agent *Agent) Close() {
	agent.Users.Close()
	agent.DeviceManager.Close()
	agent.MQTTBroker.Close()
	agent.TSDB.Close()
}
