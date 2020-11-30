package jsonrpc

import (
	"net/http"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/pkg/rpc"
)

type rpcRouter struct {
	s      *rpc.Server
	routes []router.Route
}

func NewRouter(ag *agent.Agent) router.Router {
	s := rpc.NewServer()
	if err := s.RegisterName("device", newDeviceService(ag)); err != nil {
		panic(err)
	}
	if err := s.RegisterName("alarm", newAlarmService(ag)); err != nil {
		panic(err)
	}

	r := &rpcRouter{s: s}
	r.routes = []router.Route{
		router.NewPostRoute("/rpc", r.rpc),
	}
	return r
}

func (rr *rpcRouter) Routes() []router.Route {
	return rr.routes
}

func (rr *rpcRouter) rpc(w http.ResponseWriter, r *http.Request, _ map[string]string) error {
	rr.s.ServeHTTP(w, r)
	return nil
}
