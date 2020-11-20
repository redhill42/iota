package types

// Version information contains response of remote API:
// GET "/version"
type Version struct {
	Version    string
	APIVersion string
	GitCommit  string
	BuildTime  string
	Os         string
	Arch       string
}

// Token represents an access token signed by server to
// identify a client entity.
type Token struct {
	Token string `json:"token"`
}
