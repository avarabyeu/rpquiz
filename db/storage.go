package db

type SessionRepo interface {
	Save(s *RPSession) error
	Find(dfID string) (*RPSession, error)
	Delete(dfID string) error
}

type RPSession struct {
	RpID string
	DfID string
}
