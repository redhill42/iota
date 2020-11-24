package main

// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"

	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/device"

	_ "github.com/redhill42/iota/auth/userdb/file"
	_ "github.com/redhill42/iota/auth/userdb/mongodb"
)

var users *userdb.UserDatabase
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
	// super user has full access to all topic
	if username == _SUPER_USER {
		rawSecret, err := users.GetSecret()
		if err != nil {
			fmt.Fprintf(os.Stderr, "go-auth: user database error: %v\n", err)
			return false
		}
		return password == base64.StdEncoding.EncodeToString(rawSecret)
	}

	// anonymous device can login to claim itself
	if username == "" {
		return true
	}

	if password == "" {
		// Authorized device must provide a valid token
		_, err := devices.VerifyToken(username)
		return err == nil
	}

	// Normal user must authenticate itself
	_, err := users.Authenticate(username, password)
	return err == nil
}

const (
	_MOSQ_ACL_READ      = 1
	_MOSQ_ACL_WRITE     = 2
	_MOSQ_ACL_SUBSCRIBE = 4
)

var (
	claimRequestPattern  = regexp.MustCompile("^api/v[0-9.]+/me/claim$")
	claimResponsePattern = regexp.MustCompile("^me/claim/([^/]+)$")
	apiRequestPattern    = regexp.MustCompile("^api/v[0-9.]+/([^/]+)/me/.+$")
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
		if acc == _MOSQ_ACL_WRITE {
			m := apiRequestPattern.FindStringSubmatch(topic)
			if len(m) == 2 && m[1] == username {
				return true
			}
		}

		m := apiResponsePattern.FindStringSubmatch(topic)
		return len(m) == 2 && m[1] == username
	}

	// authorized users can publish and subscribe on any topic
	return true
}

func main() {}
