package userdb

// The User interface encapsulates a cloud user. The concret User type must
// embedded a BasicUser struct that contains core information that used by
// Iota server. Extra fields may be maintained by concret User type and
// these fields will be written to the user database.
type User interface {
	// Basic returns the core information of a User.
	Basic() *BasicUser
}

// The basic user interface implementation
type BasicUser struct {
	Name     string
	Password []byte
	Inactive bool
}

func (user *BasicUser) Basic() *BasicUser {
	return user
}
