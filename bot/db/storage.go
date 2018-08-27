package db

type SessionRepo interface {
	Save(id string, s interface{}) error
	Load(id string, s interface{}) error
	Delete(id string) error
}

//type BotSession struct {
//	ID   string      `json:"id,omitempty"`
//	Data interface{} `json:"params,omitempty"`
//}
