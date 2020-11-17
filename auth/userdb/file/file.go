package file

import (
	"encoding/base64"
	"errors"
	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/config"
	"net/url"
	"os"
	"strings"
)

// User database backed by file.
type fileDB struct {
	*config.Config
}

func init() {
	prev := userdb.NewPlugin
	userdb.NewPlugin = func() (userdb.Plugin, error) {
		dbtype := config.Get("userdb.type")
		dburl := config.Get("userdb.url")

		if dbtype != "" && dbtype != "file" {
			return prev()
		}
		if dbtype == "" && !strings.HasPrefix(dburl, "file://") {
			return prev()
		}

		var filename string
		if strings.HasPrefix(dburl, "file://") {
			u, err := url.Parse(dburl)
			if err != nil {
				return nil, err
			}
			filename = u.Path
		} else {
			filename = dburl
		}
		if filename == "" {
			return nil, errors.New("User database file not configured")
		}

		conf, err := config.Open(filename)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return &fileDB{conf}, nil
	}
}

func (db *fileDB) Create(user userdb.User) error {
	basic := user.Basic()
	if db.GetOption("users", basic.Name) != "" {
		return userdb.DuplicateUserError(basic.Name)
	}
	db.AddOption("users", basic.Name, string(basic.Password))
	return db.Save()
}

func (db *fileDB) Find(name string, result userdb.User) error {
	password := db.GetOption("users", name)
	if password == "" {
		return userdb.UserNotFoundError(name)
	}
	basic := result.Basic()
	basic.Name = name
	basic.Password = []byte(password)
	return nil
}

func (db *fileDB) Search(filter interface{}, result interface{}) error {
	return userdb.Unsupported{}
}

func (db *fileDB) Remove(name string) error {
	if db.GetOption("users", name) == "" {
		return userdb.UserNotFoundError(name)
	}
	db.RemoveOption("users", name)
	return db.Save()
}

func (db *fileDB) Update(name string, fields interface{}) error {
	if db.GetOption("users", name) == "" {
		return userdb.UserNotFoundError(name)
	}
	if args, ok := fields.(userdb.Args); ok {
		if passwd, ok := args["password"]; ok {
			db.AddOption("users", name, string(passwd.([]byte)))
			return db.Save()
		}
	}
	return userdb.Unsupported{}
}

func (db *fileDB) GetSecret(key string, gen func() []byte) ([]byte, error) {
	secret := db.GetOption("secrets", key)
	if secret != "" {
		return base64.StdEncoding.DecodeString(secret)
	}

	newSecret := gen()
	db.AddOption("secrets", key, base64.StdEncoding.EncodeToString(newSecret))
	err := db.Save()
	return newSecret, err
}

func (db *fileDB) Close() error {
	return nil
}
