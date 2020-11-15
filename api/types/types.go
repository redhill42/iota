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
