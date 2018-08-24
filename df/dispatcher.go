package df

import (
	bytes2 "bytes"
	"context"
	"github.com/apex/log"
	"github.com/go-chi/render"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"io/ioutil"
	"net/http"
	"sync"
)

// ErrUnknownIntent general error to be thrown in case intent not found
var ErrUnknownIntent = errors.New("intent is unknown")

type (
	contextKey string

	//Middleware gives ability to add interceptor before intent handlers execution
	Middleware func(IntentHandler) IntentHandler

	//Dispatcher is a representation of DF API v2 engine
	Dispatcher struct {
		middlewares []Middleware
		rootHandler IntentHandler
		initSync    sync.Once
	}
)

var keyRequestRaw contextKey = "key_raw_request"

//withRawHTTPRequest sets raw http request
func withRawHTTPRequest(parent context.Context, r *http.Request) context.Context {
	return context.WithValue(parent, keyRequestRaw, r)
}

//GetRawHTTPRequest returns raw http request
func getRawHTTPRequest(ctx context.Context) *http.Request {
	msg, _ := ctx.Value(keyRequestRaw).(*http.Request)
	return msg
}

//NewIntentDispatcher creates new instance of dispatcher
func NewIntentDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// Use adds IntentHandler middleware to dispatcher
// Middlewares are executed on each request in order they have been added
func (d *Dispatcher) Use(m func(IntentHandler) IntentHandler) *Dispatcher {
	d.middlewares = append(d.middlewares, m)
	return d
}

//SetHandler sets root handler
func (d *Dispatcher) SetHandler(h IntentHandler) *Dispatcher {
	d.rootHandler = h
	return d
}

//Handle handles incoming intent request
func (d *Dispatcher) Handle(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
	rs, err := d.rootHandler.Handle(ctx, rq)
	if nil != err {
		// we do not return HTTP errors. Instead, some default error rootHandler is used to notify user
		// that something goes wrong. Such approach should simplify debugging
		log.WithError(err).Error("Intent handling error")
		rs = handleError(err)
	}
	return rs, nil
}

// HTTPHandler HTTP-based Handler for DialogFlow API v2
// Dispatches incoming request over specific intent handlers
func (d *Dispatcher) HTTPHandler() func(w http.ResponseWriter, r *http.Request) {
	d.init()
	m := jsonpb.Marshaler{}
	return func(w http.ResponseWriter, r *http.Request) {
		var wr dialogflow.WebhookRequest
		b, _ := ioutil.ReadAll(r.Body)
		//fmt.Println(string(b))
		if err := jsonpb.Unmarshal(bytes2.NewReader(b), &wr); nil != err {
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": err.Error()})
		}

		ctx := withRawHTTPRequest(r.Context(), r)
		rs, err := d.Handle(ctx, &wr)
		if nil != err {
			log.WithError(err).Error(err.Error())

			// should never be the case since errors should be handled by FallbackMiddleware
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": err.Error()})
		}

		//rs.OutputContexts = append(rs.OutputContexts, &dialogflow.Context{
		//	Name: "av",
		//	Parameters: &structpb.Struct{
		//		Fields: map[string]*structpb.Value{
		//			"rp_session": {
		//				Kind: &structpb.Value_StringValue{"123123"},
		//			},
		//		},
		//	},
		//})
		m.Marshal(w, rs)
		//render.JSON(w, r, rs)
	}
}

//initializes dispatcher's internals
func (d *Dispatcher) init() {
	d.initSync.Do(func() {
		chain := chain(d.middlewares, d.rootHandler)
		if nil == chain {
			log.Warn("Handler is not configured properly")
			chain = HandlerFunc(fallbackHandlerFunc)
		}
		d.rootHandler = chain
	})
}

// chain builds a Intent handler composed of an inline middleware stack and root
// handler in the order they are passed.
func chain(middlewares []Middleware, endpoint IntentHandler) IntentHandler {
	// Return ahead of time if there aren't any middlewares for the chain
	if len(middlewares) == 0 {
		return endpoint
	}

	// Wrap the end handler with the middleware chain
	h := middlewares[len(middlewares)-1](endpoint)
	for i := len(middlewares) - 2; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}
