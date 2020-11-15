package middleware

import "github.com/redhill42/iota/api/server/httputils"

// Middleware is an interface to allow the use of ordinary functions as API filters.
// Any struct that has the appropriate signature can be registered as a middleware.
type Middleware interface {
	WrapHandler(httputils.APIFunc) httputils.APIFunc
}
