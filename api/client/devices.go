package client

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/redhill42/iota/api/types"
)

func (api *APIClient) GetDevices(ctx context.Context) ([]string, error) {
	var devices []string
	resp, err := api.Get(ctx, "/devices", nil, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&devices)
		resp.EnsureClosed()
	}
	return devices, err
}

func (api *APIClient) GetDevice(ctx context.Context, id string, info interface{}) error {
	resp, err := api.Get(ctx, "/devices/"+id, nil, nil)
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
	resp.EnsureClosed()
	return err
}

func (api *APIClient) DeleteDevice(ctx context.Context, id string) error {
	resp, err := api.Delete(ctx, "/devices/"+id, nil, nil)
	resp.EnsureClosed()
	return err
}

func (api *APIClient) RPC(ctx context.Context, id, requestId string, request interface{}) error {
	query := url.Values{"requestId": []string{requestId}}
	resp, err := api.Post(ctx, "/devices/"+id+"/rpc", query, request, nil)
	resp.EnsureClosed()
	return err
}
