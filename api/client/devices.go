package client

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/redhill42/iota/api/types"
)

func (api *APIClient) GetDevices(ctx context.Context, keys string, result interface{}) error {
	var query url.Values
	if keys != "" {
		query = url.Values{"keys": []string{keys}}
	}

	resp, err := api.Get(ctx, "/devices", query, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(result)
		resp.EnsureClosed()
	}
	return err
}

func (api *APIClient) GetDevice(ctx context.Context, id, keys string, info interface{}) error {
	var query url.Values
	if keys != "" {
		query = url.Values{"keys": []string{keys}}
	}

	resp, err := api.Get(ctx, "/devices/"+id, query, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(info)
		resp.EnsureClosed()
	}
	return err
}

func (api *APIClient) CreateDevice(ctx context.Context, attributes interface{}) (token string, err error) {
	var v types.Token

	resp, err := api.Post(ctx, "/devices", nil, attributes, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&v)
		resp.EnsureClosed()
	}
	return v.Token, err
}

func (api *APIClient) UpdateDevice(ctx context.Context, id string, updates interface{}) error {
	resp, err := api.Put(ctx, "/devices/"+id, nil, updates, nil)
	if err == nil {
		resp.EnsureClosed()
	}
	return err
}

func (api *APIClient) DeleteDevice(ctx context.Context, id string) error {
	resp, err := api.Delete(ctx, "/devices/"+id, nil, nil)
	if err == nil {
		resp.EnsureClosed()
	}
	return err
}

func (api *APIClient) RPC(ctx context.Context, id string, request interface{}) error {
	resp, err := api.Post(ctx, "/devices/"+id+"/rpc", nil, request, nil)
	if err == nil {
		resp.EnsureClosed()
	}
	return err
}

func (api *APIClient) GetClaims(ctx context.Context) ([]map[string]interface{}, error) {
	var claims []map[string]interface{}
	resp, err := api.Get(ctx, "/claims", nil, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&claims)
		resp.EnsureClosed()
	}
	return claims, err
}

func (api *APIClient) ApproveDevice(ctx context.Context, claimId string, updates map[string]interface{}) (string, error) {
	var v types.Token

	resp, err := api.Post(ctx, "/claims/"+claimId+"/approve", nil, updates, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&v)
		resp.EnsureClosed()
	}
	return v.Token, err
}

func (api *APIClient) RejectDevice(ctx context.Context, claimId string) error {
	resp, err := api.Post(ctx, "/claims/"+claimId+"/reject", nil, nil, nil)
	if err == nil {
		resp.EnsureClosed()
	}
	return err
}
