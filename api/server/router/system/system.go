package system

import (
	"github.com/Sirupsen/logrus"
	"github.com/redhill42/iota/api"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/api/types"
	"github.com/redhill42/iota/auth"
	"net/http"
	"runtime"
)

type systemRouter struct {
	authz  *auth.Authenticator
	routes []router.Route
}

func NewRouter(authz *auth.Authenticator) router.Router {
	r := &systemRouter{authz: authz}
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

	_, token, err := s.authz.Authenticate(username, password)
	if err != nil {
		logrus.WithField("username", username).WithError(err).Debug("Login failed")
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return nil
	}

	return httputils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Token": token,
	})
}
