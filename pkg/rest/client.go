package rest

import (
	"fmt"
	"github.com/redhill42/iota/pkg/rest/transport"
	"net/http"
	"net/url"
	"strings"
)

// Client is the API client that performs all operations
// against a API server.
type Client struct {
	// proto holds the client protocol i.e. tcp.
	proto string
	// addr holds the client address.
	addr string
	// basePath holds the path to prepend to the requests.
	basePath string
	// transport is the interface to send request with, it implements transport.Client.
	transport transport.Client
	// version of the server to talk to.
	version string
	// custom http headers configured by users.
	customHTTPHeaders map[string]string
}

// NewClient initializes a new API client for the given host and API version.
// It won't send any version information if the version number is empty.
// It uses the given http client as transport.
// It also initializes the custom http headers to add to each request.
func NewClient(host string, version string, client *http.Client, httpHeaders map[string]string) (*Client, error) {
	proto, addr, basePath, err := ParseHost(host)
	if err != nil {
		return nil, err
	}

	transport, err := transport.NewTransportWithHTTP(proto, addr, client)
	if err != nil {
		return nil, err
	}

	return &Client{
		proto:             proto,
		addr:              addr,
		basePath:          basePath,
		transport:         transport,
		version:           version,
		customHTTPHeaders: httpHeaders,
	}, nil
}

// getAPIPath returns the versioned request path to call the api.
// It appends the query parameters to the path if they are not empty.
func (cli *Client) getAPIPath(p string, query url.Values) string {
	var apiPath string
	if cli.version != "" {
		v := strings.TrimPrefix(cli.version, "v")
		apiPath = fmt.Sprintf("%s/v%s%s", cli.basePath, v, p)
	} else {
		apiPath = fmt.Sprintf("%s%s", cli.basePath, p)
	}

	u := &url.URL{
		Path: apiPath,
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

// ClientVersion returns the version string associated with this instance of
// the Client.
func (cli *Client) ClientVersion() string {
	return cli.version
}

// UpdateClientVersion updates the version string associated with this
// instance of the Client.
func (cli *Client) UpdateClientVersion(v string) {
	cli.version = v
}

// Add a custom header.
func (cli *Client) AddCustomHeader(name, value string) {
	cli.customHTTPHeaders[name] = value
}

// Remove a custom header.
func (cli *Client) RemoveCustomHeader(name string) {
	delete(cli.customHTTPHeaders, name)
}

// ParseHost verifies that the given host strings is valid.
func ParseHost(host string) (string, string, string, error) {
	protoAddrParts := strings.SplitN(host, "://", 2)
	if len(protoAddrParts) == 1 {
		return "", "", "", fmt.Errorf("unable to parse host '%s'", host)
	}

	var basePath string
	proto, addr := protoAddrParts[0], protoAddrParts[1]
	if proto != "unix" {
		parsed, err := url.Parse(host)
		if err != nil {
			return "", "", "", err
		}
		addr = parsed.Host
		basePath = parsed.Path
	}
	return proto, addr, basePath, nil
}
