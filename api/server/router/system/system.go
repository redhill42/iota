package system

import (
	"net/http"
	"runtime"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/api"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/api/types"
	"github.com/sirupsen/logrus"
)

type systemRouter struct {
	*agent.Agent
	routes []router.Route
}

func NewRouter(agent *agent.Agent) router.Router {
	r := &systemRouter{Agent: agent}
	r.routes = []router.Route{
		router.NewGetRoute("/version", r.getVersion),
		router.NewGetRoute("/swagger.json", r.getSwaggerJson),
		router.NewPostRoute("/auth", r.postAuth),
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

func (s *systemRouter) postAuth(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	username, password, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Requires username and password", http.StatusUnauthorized)
		return nil
	}

	_, token, err := s.Authz.Authenticate(username, password)
	if err != nil {
		logrus.WithField("username", username).WithError(err).Debug("Login failed")
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return nil
	}

	return httputils.WriteJSON(w, http.StatusOK, types.Token{Token: token})
}
