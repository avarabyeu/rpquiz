package db

import (
	"bytes"
	"encoding/json"
	"github.com/coreos/bbolt"
	"github.com/pkg/errors"
)

var sessionBucketName = []byte("sessions")

//BoltSessionRepo represents DAO layer class for sessions DB table
type BoltSessionRepo struct {
	db *bolt.DB
}

//NewBoltSessionRepo creates new Repo instance and makes sure BoltDB bucket is also created
func NewBoltSessionRepo(db *bolt.DB) (*BoltSessionRepo, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(sessionBucketName); nil == b {
			_, err := tx.CreateBucket(sessionBucketName)
			return err
		}
		return nil
	})

	return &BoltSessionRepo{
		db: db,
	}, err

}

//Save inserts/updates entry in DB
func (r *BoltSessionRepo) Save(id string, s interface{}) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionBucketName)
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(s); nil != err {
			return err
		}
		return b.Put([]byte(id), buf.Bytes())
	})
}

//Delete removes entry from DB by its key/ID
func (r *BoltSessionRepo) Delete(dfID string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(sessionBucketName).Delete([]byte(dfID))
	})
}

//Load loads entry from DB by its ID
func (r *BoltSessionRepo) Load(id string, s interface{}) error {
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionBucketName)

		val := b.Get([]byte(id))
		if nil != val && len(val) > 0 {
			if err := json.Unmarshal(val, s); nil != err {
				return err
			}
		} else {
			//not found
			return errors.New("not found")
		}
		return nil
	})
	return err
}
