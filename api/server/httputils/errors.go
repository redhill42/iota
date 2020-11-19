package httputils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// httpStatusError is an interface that errors with custom status codes
// implement to tell the api layer which response status to set.
type httpStatusError interface {
	HTTPErrorStatusCode() int
}

type statusError struct {
	code int
	err  error
}

func (se statusError) Error() string {
	if se.err != nil {
		return se.err.Error()
	} else {
		return http.StatusText(se.code)
	}
}

func (se statusError) HTTPErrorStatusCode() int {
	return se.code
}

// NewStatusError returns a error object with HTTP status code.
func NewStatusError(code int, err error) error {
	return statusError{code, err}
}

// GetHTTPErrorStatusCode retrieve status code from error message
func GetHTTPErrorStatusCode(err error) int {
	if err == nil {
		logrus.Error("unexpected HTTP error handling")
		return http.StatusInternalServerError
	}

	var statusCode int
	errMsg := err.Error()

	if e, ok := err.(httpStatusError); ok {
		statusCode = e.HTTPErrorStatusCode()
	} else {
		// FIXME: this is brittle and should not be necessary, but we still
		// need to identify if there are errors failling back into this logic.
		// If we need to differentiate between different possible error types,
		// we should create appropriate error types that implement the
		// httpStatusError interface.
		errStr := strings.ToLower(errMsg)
		for keyword, status := range map[string]int{
			"not found":             http.StatusNotFound,
			"no such":               http.StatusNotFound,
			"bad parameter":         http.StatusBadRequest,
			"no command":            http.StatusBadRequest,
			"conflict":              http.StatusConflict,
			"impossible":            http.StatusNotAcceptable,
			"wrong login/password":  http.StatusUnauthorized,
			"unauthorized":          http.StatusUnauthorized,
			"hasn't been activated": http.StatusForbidden,
		} {
			if strings.Contains(errStr, keyword) {
				statusCode = status
				break
			}
		}
	}

	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	return statusCode
}

// WriteError decodes a specific error and sends it in the response.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil || w == nil {
		logrus.Error("unexpected HTTP error handling")
		return
	}

	statusCode := GetHTTPErrorStatusCode(err)
	serverError := fmt.Sprintf("Handler for %s %s returned error: %v", r.Method, r.URL.Path, err)

	if statusCode >= 500 {
		logrus.Error(serverError)
		http.Error(w, "Internel server error", statusCode)
	} else {
		logrus.Debug(serverError)
		http.Error(w, err.Error(), statusCode)
	}
}
