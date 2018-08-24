package db

import (
	"github.com/coreos/bbolt"
)

var sessionBucketName = []byte("sessions")

type BoltSessionRepo struct {
	db *bolt.DB
}

func (r *BoltSessionRepo) Save(s *RPSession) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionBucketName)
		return b.Put([]byte(s.DfID), []byte(s.RpID))
	})
}

func (r *BoltSessionRepo) Delete(dfID string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(sessionBucketName).Delete([]byte(dfID))
	})
}

func (r *BoltSessionRepo) Find(dfID string) (*RPSession, error) {
	var session *RPSession
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionBucketName)
		val := b.Get([]byte(dfID))
		if nil != val {
			session = &RPSession{
				RpID: string(val),
				DfID: dfID,
			}
		}
		return nil
	})
	return session, err
}

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
