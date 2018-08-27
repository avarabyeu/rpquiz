package db

//SessionRepo is a general DAO/repo interface for session entity
type SessionRepo interface {
	Save(id string, s interface{}) error
	Load(id string, s interface{}) error
	Delete(id string) error
}
