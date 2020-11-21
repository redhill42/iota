package server

import (
	"github.com/redhill42/iota/api/mqtt"
	"github.com/redhill42/iota/api/server/middleware"
	"github.com/redhill42/iota/api/server/router/devices"
	"github.com/redhill42/iota/api/server/router/system"
	"github.com/redhill42/iota/auth"
	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/device"
	"github.com/redhill42/iota/tsdb"
)

const _CONTEXT_ROOT = "/api"

// APIServer extends generic Server to serve IOTA services.
type APIServer struct {
	*Server

	users  *userdb.UserDatabase
	authz  *auth.Authenticator
	devmgr *device.DeviceManager
	broker *mqtt.Broker
	tsdb   tsdb.TSDB
}

func NewAPIServer() (api *APIServer, err error) {
	api = &APIServer{Server: New(_CONTEXT_ROOT)}

	api.users, err = userdb.Open()
	if err != nil {
		return nil, err
	}

	api.authz, err = auth.NewAuthenticator(api.users)
	if err != nil {
		return nil, err
	}

	api.devmgr, err = device.NewDeviceManager()
	if err != nil {
		return nil, err
	}

	api.broker, err = mqtt.NewBroker()
	if err != nil {
		return nil, err
	}

	api.tsdb, err = tsdb.New()
	if err != nil {
		return nil, err
	}

	// Initialize middlewares
	api.UseMiddleware(middleware.NewVersionMiddleware())
	api.UseMiddleware(middleware.NewAuthMiddleware(api.authz, api.devmgr, _CONTEXT_ROOT))

	// Initialize routers
	api.InitRouter(
		system.NewRouter(api.authz),
		devices.NewRouter(api.devmgr, api.broker, api.tsdb),
	)

	// Route MQTT request to API server.
	api.broker.Mux = api.Mux

	return api, nil
}

func (api *APIServer) Cleanup() {
	api.users.Close()
	api.devmgr.Close()
	api.broker.Close()
	api.tsdb.Close()
}
