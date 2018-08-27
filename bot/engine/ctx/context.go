package botctx

import "context"

type contextKey string

const (
	userKey         contextKey = "userKey"
	originalMessage contextKey = "originalMessage"
	session         contextKey = "session"
)

//WithUser adds a user to the context
func WithUser(ctx context.Context, u string) context.Context {
	return context.WithValue(ctx, userKey, u)
}

//GetUser takes a user from the context
func GetUser(ctx context.Context) string {
	u, ok := ctx.Value(userKey).(string)
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
func WithSession(ctx context.Context, s interface{}) context.Context {
	return context.WithValue(ctx, session, s)
}

//GetSession takes original message from the context
func GetSession(ctx context.Context) interface{} {
	return ctx.Value(session)
}
