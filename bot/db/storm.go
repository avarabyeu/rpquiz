package db

import (
	"github.com/apex/log"
	"github.com/asdine/storm"
)

//StormSessionRepo represents DAO layer class for sessions DB table
type StormSessionRepo struct {
	db *storm.DB
}

//NewStormSessionRepo creates new Repo instance and makes sure BoltDB bucket is also created
func NewStormSessionRepo(db *storm.DB) (*StormSessionRepo, error) {
	err := db.Init(&QuizSession{})
	if nil != err {
		log.Info("WTF!")
		log.Info(err.Error())
	}
	return &StormSessionRepo{
		db: db,
	}, err

}

//Save inserts/updates entry in DB
func (r *StormSessionRepo) Save(s *QuizSession) error {
	return r.db.Save(s)
}

//Update updates only non-nil fields of entry in DB
func (r *StormSessionRepo) Update(s *QuizSession) error {
	return r.db.Update(s)
}

//Delete removes entry from DB by its key/ID
func (r *StormSessionRepo) Delete(dfID string) error {
	return r.db.DeleteStruct(&QuizSession{ID: dfID})
}

//Load loads entry from DB by its ID
func (r *StormSessionRepo) Load(id string, s *QuizSession) error {
	return r.db.One("ID", id, s)
}
