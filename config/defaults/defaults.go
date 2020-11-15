package defaults

import "github.com/redhill42/iota/config"

func Domain() string {
	return config.GetOrDefault("domain", "iota.local")
}

func ApiURL() (url string) {
	return config.GetOrDefault("api.url", "http://api."+Domain())
}
