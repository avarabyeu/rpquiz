package df

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

type (
	//IntentHandler represents intent handler
	IntentHandler interface {
		Handle(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error)
	}

	// The HandlerFunc type is an adapter to allow the use of
	// ordinary functions as intent handlers.  If f is a function
	// with the appropriate signature, HandlerFunc(f) is a
	// Handler object that calls f.
	HandlerFunc func(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error)
)

// Handle calls f(ctx, req).
func (f HandlerFunc) Handle(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
	return f(ctx, rq)
}

// NewIntentHandlerFunc adding new intent rootHandler to struct
func NewIntentHandlerFunc(f func(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error)) IntentHandler {
	return HandlerFunc(f)
}

//FallbackMiddleware is a middleware that handles unknown intents and other errors
func FallbackMiddleware(next IntentHandler) IntentHandler {
	return NewIntentHandlerFunc(func(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
		rs, err := next.Handle(ctx, rq)
		if nil != err {
			if ErrUnknownIntent == err {
				return fallbackHandlerFunc(ctx, rq)
			}
			return handleError(err), nil
		}
		return rs, err
	})
}

func handleError(err error) *dialogflow.WebhookResponse {
	return NewBuilder().AddTextMessage(fmt.Sprintf("Something went wrong. Error: %s", err)).Build()
}

func fallbackHandlerFunc(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
	var text string
	if nil != rq.QueryResult {
		text = rq.QueryResult.QueryText
	} else {
		text = ""
	}
	return &dialogflow.WebhookResponse{
		FulfillmentText: fmt.Sprintf("Handler not found for '%s'. Something went wrong", text),
	}, nil
}

// NewHandlerFunc adding new intent rootHandler to struct
func NewHandlerFunc(f func(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error)) IntentHandler {
	return HandlerFunc(f)
}

//IntentNameDispatcher is a composite Handler for DialogFlow. Dispatches incoming request over specific intent handlers
func IntentNameDispatcher(handlers map[string]IntentHandler) IntentHandler {
	return NewIntentHandlerFunc(func(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
		var handler IntentHandler
		if h, ok := handlers[rq.QueryResult.Intent.DisplayName]; ok {
			handler = h
		} else {
			return nil, ErrUnknownIntent
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
