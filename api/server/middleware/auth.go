package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/auth"
	"github.com/redhill42/iota/device"
	"github.com/sirupsen/logrus"
)

type authMiddleware struct {
	authz             *auth.Authenticator
	devmgr            *device.DeviceManager
	noAuthPattern     *regexp.Regexp
	deviceAuthPattern *regexp.Regexp
}

func NewAuthMiddleware(authz *auth.Authenticator, devmgr *device.DeviceManager, contextRoot string) authMiddleware {
	return authMiddleware{
		authz,
		devmgr,
		regexp.MustCompile("^" + contextRoot + "(/v[0-9.]+)?/(version|auth|swagger.json)"),
		regexp.MustCompile("^" + contextRoot + "(/v[0-9.]+)?/me"),
	}
}

func (m authMiddleware) WrapHandler(handler httputils.APIFunc) httputils.APIFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		if m.noAuthPattern.MatchString(r.URL.Path) {
			return handler(w, r, vars)
		}

		if m.deviceAuthPattern.MatchString(r.URL.Path) {
			deviceId, err := m.devmgr.Verify(r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return nil
			}

			vars["id"] = deviceId
			return handler(w, r, vars)
		} else {
			user, err := m.authz.Verify(r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return nil
			}

			logrus.Debugf("Logged in user: %s", user.Name)
			ctx := context.WithValue(r.Context(), httputils.UserKey, user)
			return handler(w, r.WithContext(ctx), vars)
		}
	}
}
