package mongodb

import (
	"fmt"
	"reflect"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/redhill42/iota/auth/userdb"
)

// User database backed by MongoDB database.
type mongodb struct {
	session *mgo.Session
}

func mongodbPlugin(dburl string) (userdb.Plugin, error) {
	session, err := mgo.Dial(dburl)
	if err != nil {
		return nil, err
	}

	users := session.DB("").C("users")

	err = users.EnsureIndex(mgo.Index{
		Key:    []string{"name"},
		Unique: true,
	})
	if err != nil {
		session.Close()
		return nil, err
	}

	return &mongodb{session}, nil
}

func init() {
	userdb.RegisterPlugin("mongodb", mongodbPlugin)
}

func (db *mongodb) do(f func(c *mgo.Collection) error) error {
	session := db.session.Copy()
	err := f(session.DB("").C("users"))
	session.Close()
	return err
}

func (db *mongodb) Create(user userdb.User) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.Insert(user)
		if mgo.IsDup(err) {
			err = userdb.DuplicateUserError(user.Basic().Name)
		}
		return err
	})
}

func (db *mongodb) Find(name string, result userdb.User) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.Find(bson.M{"name": name}).One(result)
		if err == mgo.ErrNotFound {
			err = userdb.UserNotFoundError(name)
		}
		return err
	})
}

func (db *mongodb) Search(filter interface{}, result interface{}) error {
	return db.do(func(c *mgo.Collection) error {
		resultv := reflect.ValueOf(result)
		if resultv.Kind() == reflect.Ptr && resultv.Elem().Kind() == reflect.Slice {
			return c.Find(filter).All(result)
		} else {
			err := c.Find(filter).One(result)
			if err == mgo.ErrNotFound {
				err = userdb.UserNotFoundError(fmt.Sprintf("%v", filter))
			}
			return err
		}
	})
}

func (db *mongodb) Remove(name string) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.Remove(bson.M{"name": name})
		if err == mgo.ErrNotFound {
			err = userdb.UserNotFoundError(name)
		}
		return err
	})
}

func (db *mongodb) Update(name string, fields interface{}) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.Update(bson.M{"name": name}, bson.M{"$set": fields})
		if err == mgo.ErrNotFound {
			err = userdb.UserNotFoundError(name)
		}
		return err
	})
}

func (db *mongodb) GetSecret(key string, gen func() ([]byte, error)) ([]byte, error) {
	session := db.session.Copy()
	c := session.DB("").C("secret")
	defer session.Close()

	var record struct {
		Key    string `bson:"_id"`
		Secret []byte `bson:"secret"`
	}

	err := c.FindId(key).One(&record)
	if err == mgo.ErrNotFound {
		if record.Secret, err = gen(); err == nil {
			record.Key = key
			err = c.Insert(&record)
		}
	}
	return record.Secret, err
}

func (db *mongodb) Close() {
	db.session.Close()
}
