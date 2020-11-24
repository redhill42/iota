package main

// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"

	"github.com/redhill42/iota/auth"
	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/device"

	_ "github.com/redhill42/iota/auth/userdb/file"
	_ "github.com/redhill42/iota/auth/userdb/mongodb"
)

var users *userdb.UserDatabase
var authz *auth.Authenticator
var devices *device.Manager

const _SUPER_USER = "iota"

//export AuthPluginInit
func AuthPluginInit(keys []string, values []string, authOptsNum int) bool {
	err := config.Initialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-auth: cannot open configuration file: %v\n", err)
		return false
	}

	if users, err = userdb.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "go-auth: cannot open user database: %v\n", err)
		return false
	}

	if authz, err = auth.NewAuthenticator(users); err != nil {
		fmt.Fprintf(os.Stderr, "go-auth: cannot initialize authenticator")
		return false
	}

	if devices, err = device.NewManager(nil); err != nil {
		fmt.Fprintf(os.Stderr, "go-auth: cannot open device database: %v\n", err)
		return false
	}

	return true
}

//export AuthPluginCleanup
func AuthPluginCleanup() {
	if users != nil {
		users.Close()
	}
	if devices != nil {
		devices.Close()
	}
}

//export AuthUnpwdCheck
func AuthUnpwdCheck(username, password, clientid string) bool {
	var err error

	// super user has full access to all topic
	if username == _SUPER_USER {
		secret := authz.GetSecret()
		return password == base64.StdEncoding.EncodeToString(secret)
	}

	// anonymous devices can login to claim itself
	if username == "" {
		return true
	}

	// A user can authenticate itself with username and password
	if password != "" {
		_, err = users.Authenticate(username, password)
		return err == nil
	}

	// User can also authenticate with the access token
	if _, err = authz.VerifyToken(username); err == nil {
		return true
	}

	// Authorized device must provide a valid token
	if _, err = devices.VerifyToken(username); err == nil {
		return true
	}

	return false
}

const (
	_MOSQ_ACL_READ      = 1
	_MOSQ_ACL_WRITE     = 2
	_MOSQ_ACL_SUBSCRIBE = 4
)

var (
	claimRequestPattern  = regexp.MustCompile("^api/v[0-9.]+/me/claim$")
	claimResponsePattern = regexp.MustCompile("^me/claim/([^/]+)$")
	apiRequestPattern    = regexp.MustCompile("^api/v[0-9.]+/([^/]+)/.+$")
	apiResponsePattern   = regexp.MustCompile("^([^/]+)/me/.+$")
)

//export AuthAclCheck
func AuthAclCheck(clientid, username, topic string, acc int) bool {
	// Super user has full access to all topics
	if username == _SUPER_USER {
		return true
	}

	// anonymous device can publish request to "api/v1/me/claim" and
	// subscribe response on "me/claim/%c"
	if username == "" {
		if clientid == "" {
			return false
		}

		if acc == _MOSQ_ACL_WRITE {
			return claimRequestPattern.MatchString(topic)
		} else {
			m := claimResponsePattern.FindStringSubmatch(topic)
			return len(m) == 2 && m[1] == clientid
		}
	}

	// authorized devices can publish request to "api/v1/%u/me/attributes/request" and
	// subscribe response on "%u/me/attributes/response". It can also subscribe
	// request on "%u/me/rpc/request" and publish response to "%u/me/rpc/response"
	if _, err := devices.VerifyToken(username); err == nil {
		if m := apiRequestPattern.FindStringSubmatch(topic); len(m) == 2 {
			return acc == _MOSQ_ACL_WRITE && m[1] == username
		}

		if m := apiResponsePattern.FindStringSubmatch(topic); len(m) == 2 {
			return m[1] == username
		}

		// devices can communicate each other on any topics
		return true
	}

	// authorized users has full access on any topic
	return true
}

func main() {}
