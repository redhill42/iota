package client

import (
	"github.com/redhill42/iota/pkg/rest"
	"net/http"
)

type APIClient struct {
	cli *rest.Client
}

func NewAPIClient(host, version string, client *http.Client, httpHeaders map[string]string) (*APIClient, error) {
	cli, err := rest.NewClient(host, version, client, httpHeaders)
	if err != nil {
		return nil, err
	}
	return &APIClient{cli}, nil
}
