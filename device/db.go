package device

import (
	"crypto/rand"
	"errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"sync"

	"github.com/redhill42/iota/config"
)

type deviceDB struct {
	session *mgo.Session
	cache   sync.Map
}

func openDatabase() (*deviceDB, error) {
	dburl := config.Get("devicedb.url")
	if dburl == "" {
		return nil, errors.New("Device database URL not configured")
	}

	session, err := mgo.Dial(dburl)
	if err != nil {
		return nil, err
	}
	return &deviceDB{session: session}, nil
}

func (db *deviceDB) do(f func(c *mgo.Collection) error) error {
	session := db.session.Copy()
	err := f(session.DB("").C("devices"))
	session.Close()
	return err
}

func (db *deviceDB) Create(id, token string, attributes map[string]interface{}) error {
	return db.do(func(c *mgo.Collection) error {
		delete(attributes, "id")
		delete(attributes, "token")
		attributes["_id"] = id
		attributes["_token"] = token

		err := c.Insert(attributes)
		if mgo.IsDup(err) {
			err = DuplicateDeviceError(id)
		}
		return err
	})
}

func (db *deviceDB) Find(id string, result map[string]interface{}) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.FindId(id).One(result)
		if err == mgo.ErrNotFound {
			err = DeviceNotFoundError(id)
		}
		if err != nil {
			return err
		}

		token := result["_token"]
		delete(result, "_id")
		delete(result, "_token")
		result["id"] = id
		result["token"] = token
		return nil
	})
}

func (db *deviceDB) FindAll() ([]string, error) {
	var result []string
	err := db.do(func(c *mgo.Collection) error {
		var v struct {
			Id string `bson:"_id"`
		}

		iter := c.Find(nil).Select(bson.M{"_id": 1}).Iter()
		for iter.Next(&v) {
			result = append(result, v.Id)
		}
		return iter.Close()
	})
	return result, err
}

func (db *deviceDB) GetToken(id string) (string, error) {
	if token, ok := db.cache.Load(id); ok {
		return token.(string), nil
	}

	var v struct {
		Token string `bson:"_token"`
	}
	err := db.do(func(c *mgo.Collection) error {
		err := c.FindId(id).Select(bson.M{"_token": 1}).One(&v)
		if err == mgo.ErrNotFound {
			err = DeviceNotFoundError(id)
		}
		return err
	})
	if err == nil {
		db.cache.Store(id, v.Token)
	}
	return v.Token, err
}

func (db *deviceDB) Update(id string, fields map[string]interface{}) error {
	return db.do(func(c *mgo.Collection) error {
		delete(fields, "_id")
		delete(fields, "_token")
		if len(fields) == 0 {
			return nil
		}

		err := c.UpdateId(id, bson.M{"$set": fields})
		if err == mgo.ErrNotFound {
			err = DeviceNotFoundError(id)
		}
		return err
	})
}

func (db *deviceDB) Upsert(id, token string, fields map[string]interface{}) error {
	return db.do(func(c *mgo.Collection) error {
		delete(fields, "id")
		delete(fields, "_id")
		delete(fields, "token")
		fields["_token"] = token

		_, err := c.UpsertId(id, bson.M{"$set": fields})
		return err
	})
}

func (db *deviceDB) Remove(id string) error {
	return db.do(func(c *mgo.Collection) error {
		db.cache.Delete(id)
		err := c.RemoveId(id)
		if err == mgo.ErrNotFound {
			err = DeviceNotFoundError(id)
		}
		return err
	})
}

func (db *deviceDB) getSecret(key string) ([]byte, error) {
	session := db.session.Copy()
	c := session.DB("").C("secret")
	defer session.Close()

	var record struct {
		Key    string `bson:"_id"`
		Secret []byte `bson:"secret"`
	}

	err := c.FindId(key).One(&record)
	if err == mgo.ErrNotFound {
		record.Key = key
		record.Secret = make([]byte, 64)
		rand.Read(record.Secret)
		err = c.Insert(&record)
	}
	return record.Secret, err
}

func (db *deviceDB) Close() {
	db.session.Close()
}
