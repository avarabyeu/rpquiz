package bot

import (
	"context"
	"github.com/pkg/errors"
)

// ErrUnknownIntent general error to be thrown in case intent not found
var ErrUnknownIntent = errors.New("intent is unknown")

const acceptableConfidence = 0.5

//IntentNameDispatcher is a composite Handler for DialogFlow. Dispatches incoming request over specific intent handlers
func IntentNameDispatcher(intentHandlers map[string]Handler, callbackHandler Handler, fallback Handler) Handler {
	return HandlerFunc(func(ctx context.Context, rq Request) ([]*Response, error) {
		var handler Handler

		switch irq := rq.(type) {
		case *IntentRequest:

			if irq.Confidence < acceptableConfidence {
				//intent isn't recognized. No reason to search for intent handler at all
				handler = fallback
			} else if h, ok := intentHandlers[irq.Intent]; ok {
				//intent is recognized and handler is implemented
				handler = h
			} else {
				//intent is recognized but handler not implemented
				handler = fallback
			}

		case *CallbackRequest:
			handler = callbackHandler
		default:
			//intent is recognized but handler not implemented
			handler = callbackHandler

		}

		return handler.Handle(ctx, rq)
	})
}
