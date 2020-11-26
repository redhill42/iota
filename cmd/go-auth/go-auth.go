package main

// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"

import (
	"github.com/redhill42/iota/mqtt/mosquitto"
)

//export AuthPluginInit
func AuthPluginInit(keys []string, values []string, authOptsNum int) bool {
	return mosquitto.AuthPluginInit(keys, values, authOptsNum)
}

//export AuthPluginCleanup
func AuthPluginCleanup() {
	mosquitto.AuthPluginCleanup()
}

//export AuthUnpwdCheck
func AuthUnpwdCheck(username, password, clientid string) bool {
	return mosquitto.AuthUnpwdCheck(username, password, clientid)
}

//export AuthAclCheck
func AuthAclCheck(clientid, username, topic string, acc int) bool {
	return mosquitto.AuthAclCheck(clientid, username, topic, acc)
}

func main() {}
