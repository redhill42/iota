package userdb

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/url"
	"strings"

	"github.com/redhill42/iota/config"
)

// The Plugin interface represents a user database plugin. This interface
// provides CRUD operations for users. The user database can be backed by
// relational or NoSQL database, LDAP or Kerberos services.
type Plugin interface {
	// Create a new use in the database
	Create(user User) error

	// Find the user by name.
	Find(name string, result User) error

	// Search user database by the given filter.
	Search(filter interface{}, result interface{}) error

	// Remove the user from the database.
	Remove(name string) error

	// Update user with the new data
	Update(name string, fields interface{}) error

	// GetSecret returns a secret key used to sign the JWT token. If the
	// secret key does not exist in the database, a new key is generated
	// and saved to the database.
	GetSecret(key string, gen func() []byte) ([]byte, error)

	// Close the user database.
	Close()
}

// PluginFunc represents a plugin initialization function.
type PluginFunc func(dburl string) (Plugin, error)

var pluginRegistration = make(map[string]PluginFunc)

// RegisterPlugin register a plugin under the given database scheme.
func RegisterPlugin(scheme string, f PluginFunc) {
	pluginRegistration[scheme] = f
}

// NewPlugin create a new plugin according to the configured user database URL.
func NewPlugin() (Plugin, error) {
	dbtype := config.Get("userdb.type")
	dburl := config.Get("userdb.url")

	if dbtype == "" && dburl != "" {
		u, err := url.Parse(dburl)
		if err != nil {
			return nil, err
		}
		dbtype = u.Scheme
	}

	if dbtype == "" {
		return nil, errors.New("The user database plugin does not configured")
	} else if f, ok := pluginRegistration[dbtype]; ok {
		return f(dburl)
	} else {
		return nil, fmt.Errorf("Unsupported user database scheme: %s", dbtype)
	}
}

// Utility type to create filters and update fields
type Args map[string]interface{}

// The DuplicateUserError indicates that an user already exists in the database
// when creating user.
type DuplicateUserError string

// The UserNotFoundError indicates that a user not found in the database.
type UserNotFoundError string

// The InactiveUserError indicates that a user is not valid to login.
type InactiveUserError string

func (e DuplicateUserError) Error() string {
	return fmt.Sprintf("User already exists: %s", string(e))
}

func (e DuplicateUserError) HTTPErrorStatusCode() int {
	return http.StatusConflict
}

func (e UserNotFoundError) Error() string {
	return fmt.Sprintf("User not found: %s", string(e))
}

func (e UserNotFoundError) HTTPErrorStatusCode() int {
	return http.StatusNotFound
}

func (e InactiveUserError) Error() string {
	return fmt.Sprintf("You cannot login using this identity: %s", string(e))
}

func (e InactiveUserError) HTTPErrorStatusCode() int {
	return http.StatusUnauthorized
}

type Unsupported struct {
}

func (e Unsupported) Error() string {
	return "Unsupported operation"
}

func (e Unsupported) HTTPErrorStatusCode() int {
	return http.StatusInternalServerError
}

// The UserDatabase type is the central point of user management.
type UserDatabase struct {
	plugin Plugin
}

func Open() (*UserDatabase, error) {
	plugin, err := NewPlugin()
	if err != nil {
		return nil, err
	}
	return &UserDatabase{plugin}, nil
}

func (db *UserDatabase) Create(user User, password string) error {
	basic := user.Basic()

	if basic.Name == "" || len(password) == 0 {
		return errors.New("missing required parameters")
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}

	basic.Inactive = false
	basic.Password = hashedPassword
	return db.plugin.Create(user)
}

func hashPassword(password string) ([]byte, error) {
	// use the password if it's already hashed
	if strings.HasPrefix(password, "$2a$") {
		if _, err := bcrypt.Cost([]byte(password)); err == nil {
			return []byte(password), nil
		}
	}

	// otherwise, generate a hashed password
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func (db *UserDatabase) Find(name string, result User) error {
	return db.plugin.Find(name, result)
}

func (db *UserDatabase) Search(filter interface{}, result interface{}) error {
	return db.plugin.Search(filter, result)
}

func (db *UserDatabase) Remove(name string) error {
	return db.plugin.Remove(name)
}

func (db *UserDatabase) Update(name string, fields interface{}) error {
	return db.plugin.Update(name, fields)
}

func (db *UserDatabase) Authenticate(name string, password string) (*BasicUser, error) {
	var user BasicUser
	if err := db.plugin.Find(name, &user); err != nil {
		return nil, err
	}

	if user.Inactive {
		return nil, InactiveUserError(name)
	}

	err := bcrypt.CompareHashAndPassword(user.Password, []byte(password))
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *UserDatabase) ChangePassword(name string, oldPassword, newPassword string) error {
	var user BasicUser
	if err := db.plugin.Find(name, &user); err != nil {
		return err
	}

	err := bcrypt.CompareHashAndPassword(user.Password, []byte(oldPassword))
	if err != nil {
		return err
	}

	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	return db.plugin.Update(name, Args{"password": hashedPassword})
}

// GetSecret returns a secret key used to sign the JWT token. If the
// secret key does not exist in the database, a new key is generated
// and saved to the database.
func (db *UserDatabase) GetSecret(key string, gen func() []byte) ([]byte, error) {
	return db.plugin.GetSecret(key, gen)
}

func (db *UserDatabase) Close() {
	db.plugin.Close()
}
