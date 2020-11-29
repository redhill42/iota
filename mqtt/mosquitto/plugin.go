package mosquitto

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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

const superUser = "iota"

var superUserPw string

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

	password, err := users.GetPassword("mqtt", 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-auth: user database error: %v\n", err)
		return false
	}
	superUserPw = string(password)

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

func AuthPluginCleanup() {
	if users != nil {
		users.Close()
	}
	if devices != nil {
		devices.Close()
	}
}

func AuthUnpwdCheck(username, password, clientid string) bool {
	var err error

	// super user has full access to all topic
	if username == superUser {
		return password == superUserPw
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

func AuthAclCheck(clientid, username, topic string, acc int) bool {
	// Super user has full access to all topics
	if username == superUser {
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

	if _, err := devices.VerifyToken(username); err == nil {
		// authorized device can publish request to api request topic, either
		// for itself or other devices
		if m := apiRequestPattern.FindStringSubmatch(topic); len(m) == 2 {
			if acc == _MOSQ_ACL_WRITE {
				if m[1] == username {
					return true
				} else {
					_, err = devices.VerifyToken(m[1])
					return err == nil
				}
			}
			return false
		}

		// device can subscribe api response topic for itself or other devices
		if m := apiResponsePattern.FindStringSubmatch(topic); len(m) == 2 {
			if m[1] == username {
				return true
			} else {
				_, err = devices.VerifyToken(m[1])
				return err == nil
			}
		}

		// check for wildcard topics
		if acc == _MOSQ_ACL_SUBSCRIBE {
			if topic == "#" || strings.HasPrefix(topic, "api/") || strings.HasPrefix(topic, "+/") {
				return false
			}
		}

		// devices can communicate each other on any topics
		return true
	}

	// authorized users has full access on any topic
	return true
}
