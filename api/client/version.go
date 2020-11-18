package client

import (
	"context"
	"encoding/json"
	"runtime"

	version_lib "github.com/redhill42/iota/api"
	"github.com/redhill42/iota/api/types"
)

func (api *APIClient) ServerVersion(ctx context.Context) (types.Version, error) {
	var server types.Version
	resp, err := api.Get(ctx, "/version", nil, nil)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&server)
		resp.EnsureClosed()
	}
	return server, err
}

func (api *APIClient) ClientVersion() types.Version {
	return types.Version{
		Version:    version_lib.Version,
		APIVersion: api.Client.ClientVersion(),
		GitCommit:  version_lib.GitCommit,
		BuildTime:  version_lib.BuildTime,
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}
}
