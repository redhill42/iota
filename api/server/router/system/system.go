package system

import (
	"github.com/redhill42/iota/api"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/api/types"
	"net/http"
	"runtime"
)

type systemRouter struct {
	routes []router.Route
}

func NewRouter() router.Router {
	r := &systemRouter{}
	r.routes = []router.Route{
		router.NewGetRoute("/version", r.getVersion),
		router.NewGetRoute("/swagger.json", r.getSwaggerJson),
	}
	return r
}

func (s *systemRouter) Routes() []router.Route {
	return s.routes
}

func (s *systemRouter) getVersion(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	v := types.Version{
		Version:    api.Version,
		APIVersion: api.APIVersion,
		GitCommit:  api.GitCommit,
		BuildTime:  api.BuildTime,
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}
	return httputils.WriteJSON(w, http.StatusOK, v)
}
