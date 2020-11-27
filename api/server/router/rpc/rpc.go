package rpc

import (
	rpcserver "github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
	"net/http"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/api/server/router"
)

type rpcRouter struct {
	s      *rpcserver.Server
	routes []router.Route
}

func NewRouter(ag *agent.Agent) router.Router {
	s := rpcserver.NewServer()
	s.RegisterCodec(json2.NewCodec(), "application/json")
	s.RegisterService(newDeviceService(ag), "device")

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
