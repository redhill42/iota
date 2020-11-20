package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/redhill42/iota/api/types"
)

func (api *APIClient) Authenticate(ctx context.Context, username, password string) (token string, err error) {
	var v types.Token

	auth := string(base64.StdEncoding.EncodeToString([]byte(username + ":" + password)))
	headers := map[string][]string{"Authorization": {"Basic " + auth}}
	resp, err := api.Post(ctx, "/auth", nil, nil, headers)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&v)
		resp.EnsureClosed()
	}
	return v.Token, err
}

func (api *APIClient) SetToken(token string) {
	if token != "" {
		api.AddCustomHeader("Authorization", "bearer "+token)
	} else {
		api.RemoveCustomHeader("Authorization")
	}
}
