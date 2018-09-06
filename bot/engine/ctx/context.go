package botctx

import (
	"context"
	"github.com/avarabyeu/rpquiz/bot/db"
)

type contextKey string

const (
	userNameKey     contextKey = "userNameKey"
	userIDKey       contextKey = "userIDKey"
	originalMessage contextKey = "originalMessage"
	session         contextKey = "session"
)

//WithUserName adds a user name to the context
func WithUserName(ctx context.Context, u string) context.Context {
	return context.WithValue(ctx, userNameKey, u)
}

//GetUserName takes a user from the context
func GetUserName(ctx context.Context) string {
	u, ok := ctx.Value(userNameKey).(string)
	if !ok {
		return ""
	}
	return u
}

//WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, u string) context.Context {
	return context.WithValue(ctx, userIDKey, u)
}

//GetUserID takes a user ID from the context
func GetUserID(ctx context.Context) string {
	u, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return ""
	}
	return u
}

//WithOriginalMessage adds original message to the context
func WithOriginalMessage(ctx context.Context, msg interface{}) context.Context {
	return context.WithValue(ctx, originalMessage, msg)
}

//GetOriginalMessage takes original message from the context
func GetOriginalMessage(ctx context.Context) interface{} {
	return ctx.Value(originalMessage)
}

//WithSession adds original message to the context
func WithSession(ctx context.Context, s *db.QuizSession) context.Context {
	return context.WithValue(ctx, session, s)
}

//GetSession takes original message from the context
func GetSession(ctx context.Context) (*db.QuizSession, bool) {
	if val := ctx.Value(session); nil != val {
		s, ok := val.(*db.QuizSession)
		return s, ok
	}
	return nil, false

}
