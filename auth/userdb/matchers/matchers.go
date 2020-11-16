package matchers

import (
	"github.com/onsi/gomega/format"
	"github.com/redhill42/iota/auth/userdb"
)

type BeDuplicateUser string

func (matcher BeDuplicateUser) Match(actual interface{}) (success bool, err error) {
	actualErr, ok := actual.(userdb.DuplicateUserError)
	return ok && string(actualErr) == string(matcher), nil
}

func (matcher BeDuplicateUser) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be a", userdb.DuplicateUserError(string(matcher)))
}

func (matcher BeDuplicateUser) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be a", userdb.DuplicateUserError(string(matcher)))
}

type BeUserNotFound string

func (matcher BeUserNotFound) Match(actual interface{}) (success bool, err error) {
	actualErr, ok := actual.(userdb.UserNotFoundError)
	return ok && (string(matcher) == "" || string(matcher) == string(actualErr)), nil
}

func (matcher BeUserNotFound) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be a", userdb.UserNotFoundError(string(matcher)))
}

func (matcher BeUserNotFound) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be a ", userdb.UserNotFoundError(string(matcher)))
}
