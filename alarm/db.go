package alarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/redhill42/iota/config"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Severity byte
type Status byte

const (
	Critical Severity = iota
	Major
	Minor
	Warning
)

const (
	Active Status = iota
	Cleared
)

type Alarm struct {
	ID          string                 `json:"id" bson:"-"`
	Name        string                 `json:"name"`
	Originator  string                 `json:"originator"`
	Severity    Severity               `json:"severity"`
	Status      Status                 `json:"status"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	UpdateTime  time.Time              `json:"updateTime"`
	ClearTime   time.Time              `json:"clearTime"`
}

func (a *Alarm) GetID() string {
	return a.ID
}

func (a *Alarm) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

type alarmKey struct {
	Name       string
	Originator string
}

type alarmID struct {
	bson.ObjectId `bson:"_id"`
}

type alarmRec struct {
	ID    bson.ObjectId `bson:"_id"`
	Alarm `bson:",inline"`
}

type NotFoundError string

func (e NotFoundError) Error() string {
	return fmt.Sprintf("Alarm not found: %s", string(e))
}

func (e NotFoundError) HTTPErrorStatusCode() int {
	return http.StatusNotFound
}

type alarmDB struct {
	session *mgo.Session
}

func openDatabase() (*alarmDB, error) {
	dburl := config.Get("devicedb.url") // Reuse device database
	if dburl == "" {
		return nil, errors.New("Device database URL not configured")
	}

	session, err := mgo.Dial(dburl)
	if err != nil {
		return nil, err
	}

	alarms := session.DB("").C("alarms")
	err = alarms.EnsureIndex(mgo.Index{
		Key:    []string{"name", "originator"},
		Unique: true,
	})
	if err != nil {
		session.Close()
		return nil, err
	}

	return &alarmDB{session}, nil
}

func (db *alarmDB) do(f func(c *mgo.Collection) error) error {
	session := db.session.Copy()
	defer session.Close()
	return f(session.DB("").C("alarms"))
}

func (db *alarmDB) Upsert(alarm *Alarm) error {
	return db.do(func(c *mgo.Collection) error {
		alarm.Status = Active
		alarm.UpdateTime = time.Now()
		alarm.ClearTime = time.Time{}

		key := alarmKey{alarm.Name, alarm.Originator}
		info, err := c.Upsert(key, alarm)
		if err != nil {
			return err
		}

		var id alarmID
		if info.UpsertedId != nil {
			id.ObjectId = info.UpsertedId.(bson.ObjectId)
		} else {
			err = c.Find(key).Select(bson.M{"_id": 1}).One(&id)
		}
		if err == nil {
			alarm.ID = id.Hex()
		}
		return err
	})
}

func (db *alarmDB) Delete(id string) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.RemoveId(bson.ObjectIdHex(id))
		if err == mgo.ErrNotFound {
			err = NotFoundError(id)
		}
		return err
	})
}

func (db *alarmDB) DeleteName(name, originator string) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.Remove(alarmKey{name, originator})
		if err == mgo.ErrNotFound {
			err = NotFoundError(name)
		}
		return err
	})
}

func (db *alarmDB) Clear(id string) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.UpdateId(bson.ObjectIdHex(id),
			bson.M{"$set": bson.M{"status": Cleared, "cleartime": time.Now()}})
		if err == mgo.ErrNotFound {
			err = NotFoundError(id)
		}
		return err
	})
}

func (db *alarmDB) ClearName(name, originator string) error {
	return db.do(func(c *mgo.Collection) error {
		err := c.Update(alarmKey{name, originator},
			bson.M{"$set": bson.M{"status": Cleared, "cleartime": time.Now()}})
		if err == mgo.ErrNotFound {
			err = NotFoundError(name)
		}
		return err
	})
}

func (db *alarmDB) Find(id string) (*Alarm, error) {
	alarm := Alarm{ID: id}
	err := db.do(func(c *mgo.Collection) error {
		err := c.FindId(bson.ObjectIdHex(id)).One(&alarm)
		if err == mgo.ErrNotFound {
			err = NotFoundError(id)
		}
		return err
	})
	return &alarm, err
}

func (db *alarmDB) FindName(name, originator string) (*Alarm, error) {
	var rec alarmRec
	err := db.do(func(c *mgo.Collection) error {
		err := c.Find(alarmKey{name, originator}).One(&rec)
		if err == mgo.ErrNotFound {
			err = NotFoundError(name)
		}
		rec.Alarm.ID = rec.ID.Hex()
		return err
	})
	return &rec.Alarm, err
}

func (db *alarmDB) FindAll() ([]*Alarm, error) {
	rec := make([]alarmRec, 0)
	err := db.do(func(c *mgo.Collection) error {
		return c.Find(bson.M{}).All(&rec)
	})
	if err != nil {
		return nil, err
	}
	result := make([]*Alarm, len(rec))
	for i := 0; i < len(rec); i++ {
		result[i] = &rec[i].Alarm
		result[i].ID = rec[i].ID.Hex()
	}
	return result, nil
}

func (db *alarmDB) Close() {
	db.session.Close()
}
