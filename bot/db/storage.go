package db

//SessionRepo is a general DAO/repo interface for session entity
type SessionRepo interface {
	Save(s *QuizSession) error
	Load(id string, s *QuizSession) error
	Delete(id string) error
	Update(s *QuizSession) error
}
