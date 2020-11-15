package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/redhill42/iota/api"
	"github.com/redhill42/iota/api/server/httputils"
)

type badRequestError struct {
	error
}

func (badRequestError) HTTPErrorStatusCode() int {
	return http.StatusBadRequest
}

// VersionMiddleware is a middleware that validates the client and server versions.
type VersionMiddleware struct {
}

// NewVersionMiddleware creates a new VersionMiddleware with the default versions
func NewVersionMiddleware() VersionMiddleware {
	return VersionMiddleware{}
}

// WrapHandler returns a new handler function wrapping the previous one in the request chain
func (m VersionMiddleware) WrapHandler(handler httputils.APIFunc) httputils.APIFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		apiVersion := vars["version"]
		if apiVersion == "" {
			apiVersion = api.APIVersion
		}

		if api.CompareVersions(apiVersion, api.APIVersion) > 0 {
			return badRequestError{
				fmt.Errorf("client is newer than server (client API version: %s, server API version: %s)",
					apiVersion, api.APIVersion),
			}
		}
		if api.CompareVersions(apiVersion, api.MinAPIVersion) < 0 {
			return badRequestError{
				fmt.Errorf("client version %s is too old, Minimum supported API version is %s, "+
					"please upgrade your client to a newer version", apiVersion, api.MinAPIVersion),
			}
		}

		header := fmt.Sprintf("IOTA-API/%s", api.Version)
		w.Header().Set("Server", header)
		ctx := context.WithValue(r.Context(), httputils.APIVersionKey, apiVersion)
		return handler(w, r.WithContext(ctx), vars)
	}
}
