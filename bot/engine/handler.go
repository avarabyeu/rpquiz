package bot

import (
	"context"
	"github.com/apex/log"
	"github.com/pkg/errors"
)

// ErrUnknownIntent general error to be thrown in case intent not found
var ErrUnknownIntent = errors.New("intent is unknown")

const acceptableConfidence = 0.5

//IntentNameDispatcher is a composite Handler for DialogFlow. Dispatches incoming request over specific intent handlers
func IntentNameDispatcher(handlers map[string]Handler, fallback Handler) Handler {
	return HandlerFunc(func(ctx context.Context, rq *Request) (*Response, error) {
		var handler Handler

		if rq.Confidence < acceptableConfidence {
			//intent isn't recognized. No reason to search for intent handler at all
			handler = fallback
		} else if h, ok := handlers[rq.Intent]; ok {
			//intent is recognized and handler is implemented
			handler = h
		} else {
			//intent is recognized but handler not implemented
			handler = fallback
		}

		rs, err := handler.Handle(ctx, rq)
		if nil != err {
			// we do not return HTTP errors. Instead, some special error handler is used to notify user
			// that something goes wrong. Such approach should simplify debugging
			log.WithError(err).Error("Error on intent handling")
		}
		return rs, err
	})
}
