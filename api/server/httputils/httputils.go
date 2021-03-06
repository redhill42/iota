package httputils

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/redhill42/iota/auth/userdb"
	"github.com/sirupsen/logrus"
)

// key is an unexported type for keys defined in this package
// This prevents collisions with keys defined in other packages.
type key int

// APIVersionKey is the client's requested API version
const APIVersionKey key = 0

// UserKey is the key for userdb.User values in contexts.
const UserKey key = 1

// APIFunc is an adapter to allow the use of ordinary functions as API endpoints.
// Any function that has the appropriate signature can be registered as an API endpoint.
type APIFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) error

// CheckForJSON makes sure that the request's Content-Type is application/json.
func CheckForJSON(r *http.Request) error {
	ct := r.Header.Get("Content-Type")

	// No Content-Type header is ok as long as there's no body
	if ct == "" {
		if r.Body == nil || r.ContentLength == 0 {
			return nil
		}
	}

	// Otherwise it better be json
	if MatchesContentType(ct, "application/json") {
		return nil
	}
	return fmt.Errorf("Content-Type specified (%s) must be 'application/json'", ct)
}

// ReadJSON reads the value from http request stream as json with standard json encoding.
func ReadJSON(r *http.Request, v interface{}) (err error) {
	if err = ParseForm(r); err != nil {
		return NewStatusError(http.StatusBadRequest, err)
	}
	if err = CheckForJSON(r); err != nil {
		return NewStatusError(http.StatusBadRequest, err)
	}
	if err = json.NewDecoder(r.Body).Decode(v); err != nil {
		return NewStatusError(http.StatusBadRequest, err)
	}
	return nil
}

// WriteJSON writes the value to the http response stream as json with standard json encoding.
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

// ParseForm ensures the request form is parsed even with invalid content types.
// If we don't do this, POST method without Content-Type (even with empty body) will fail.
func ParseForm(r *http.Request) error {
	if r == nil {
		return nil
	}
	if err := r.ParseForm(); err != nil && !strings.HasPrefix(err.Error(), "mime:") {
		return err
	}
	return nil
}

// MatchesContentType validates the content type against the expected one
func MatchesContentType(contentType, expectedType string) bool {
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logrus.WithError(err).Errorf("Error parsing media type: %s", contentType)
	}
	return err == nil && mimeType == expectedType
}

// VersionFromContext returns an API version from the context using APIVersionKey.
func VersionFromContext(ctx context.Context) (ver string) {
	if ctx == nil {
		return
	}
	val := ctx.Value(APIVersionKey)
	if val == nil {
		return
	}
	return val.(string)
}

// UserFromContext returns the authenticated user from the context using UserKey.
func UserFromContext(ctx context.Context) (user *userdb.BasicUser) {
	if ctx == nil {
		return
	}
	val := ctx.Value(UserKey)
	if val == nil {
		return
	}
	return val.(*userdb.BasicUser)
}
