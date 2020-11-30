package device

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"sync"

	"github.com/redhill42/iota/config"
)

type deviceDB struct {
	session *mgo.Session
	cache   sync.Map
}

type selector bson.M

func newSelector(keys []string) selector {
	sel := make(selector)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			if key == "id" {
				key = "_id"
			} else if key == "token" {
				key = "_token"
			}
			sel[key] = 1
		}
	}
	return sel
}

func (sel selector) contains(key string) bool {
	if len(sel) == 0 {
		return true
	} else {
		_, ok := sel[key]
		return ok
	}
}

type Record map[string]interface{}

func (r Record) afterLoad(sel selector) {
	if id, ok := r["_id"]; ok {
		delete(r, "_id")
		if sel.contains("_id") {
			r["id"] = id
		}
	}
	if tok, ok := r["_token"]; ok {
		delete(r, "_token")
		r["token"] = tok
	}
}

func (r Record) GetID() string {
	return r["id"].(string)
}

func (r Record) Marshal() ([]byte, error) {
	return json.Marshal(r)
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

func (db *deviceDB) Create(id, token string, attributes Record) error {
	if !validateDeviceId(id) {
		return InvalidDeviceIdError(id)
	}
	if attributes == nil {
		attributes = make(Record)
	}
	delete(attributes, "id")
	delete(attributes, "token")
	attributes["_id"] = id
	attributes["_token"] = token

	return db.do(func(c *mgo.Collection) error {
		err := c.Insert(attributes)
		if mgo.IsDup(err) {
			err = DuplicateDeviceError(id)
		}
		return err
	})
}

func (db *deviceDB) Find(id string, keys []string) (result Record, err error) {
	err = db.do(func(c *mgo.Collection) error {
		var sel selector
		if len(keys) == 0 {
			err = c.FindId(id).One(&result)
		} else {
			sel = newSelector(keys)
			err = c.FindId(id).Select(sel).One(&result)
		}
		if err == mgo.ErrNotFound {
			err = DeviceNotFoundError(id)
		}
		result.afterLoad(sel)
		return err
	})
	return
}

func (db *deviceDB) FindAll(keys []string) (result []Record, err error) {
	err = db.do(func(c *mgo.Collection) error {
		var iter *mgo.Iter
		var sel selector
		var record Record

		if len(keys) == 0 {
			iter = c.Find(nil).Iter()
		} else {
			sel = newSelector(keys)
			iter = c.Find(nil).Select(sel).Iter()
		}
		for iter.Next(&record) {
			record.afterLoad(sel)
			result = append(result, record)
			record = nil
		}
		return iter.Close()
	})
	return
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

func (db *deviceDB) Update(id string, fields Record) error {
	delete(fields, "_id")
	delete(fields, "id")
	delete(fields, "_token")
	delete(fields, "token")
	if len(fields) == 0 {
		return nil
	}

	return db.do(func(c *mgo.Collection) (err error) {
		var remove []string
		for k, v := range fields {
			if v == nil {
				delete(fields, k)
				remove = append(remove, k)
			}
		}

		if len(remove) == 0 {
			err = c.UpdateId(id, bson.M{"$set": fields})
		} else if len(fields) == 0 {
			err = c.UpdateId(id, []bson.M{{"$unset": remove}})
		} else {
			err = c.UpdateId(id, []bson.M{{"$set": fields}, {"$unset": remove}})
		}
		if err == mgo.ErrNotFound {
			err = DeviceNotFoundError(id)
		}
		return err
	})
}

func (db *deviceDB) Upsert(id, token string, fields Record) error {
	if !validDeviceIdPattern.MatchString(id) {
		return InvalidDeviceIdError(id)
	}
	if fields == nil {
		fields = make(Record)
	}
	delete(fields, "id")
	delete(fields, "_id")
	delete(fields, "token")
	fields["_token"] = token

	return db.do(func(c *mgo.Collection) error {
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
