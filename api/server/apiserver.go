package server

import (
	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/api/server/middleware"
	"github.com/redhill42/iota/api/server/router/alarms"
	"github.com/redhill42/iota/api/server/router/devices"
	"github.com/redhill42/iota/api/server/router/jsonrpc"
	"github.com/redhill42/iota/api/server/router/system"
)

const _CONTEXT_ROOT = "/api"

func NewAPIServer(agent *agent.Agent) (api *Server, err error) {
	api = New(_CONTEXT_ROOT)

	// Initialize middlewares
	api.UseMiddleware(middleware.NewVersionMiddleware())
	api.UseMiddleware(middleware.NewAuthMiddleware(agent, _CONTEXT_ROOT))

	// Initialize routers
	api.InitRouter(
		system.NewRouter(agent),
		jsonrpc.NewRouter(agent),
		devices.NewRouter(agent),
		alarms.NewRouter(agent),
	)

	// Forward MQTT request to API server.
	err = agent.MQTTBroker.Forward(api.Mux)

	return api, err
}
