package bot

import (
	"context"
	"github.com/apex/log"
	"github.com/pkg/errors"
	"gitlab.com/avarabyeu/rpquiz/bot/nlp"
	"sync"
)

type (
	//Handler handles some particular user intent
	Handler interface {
		Handle(ctx context.Context, rq *Request) ([]*Response, error)
	}
	//ErrorHandler converts error to human-readable response
	ErrorHandler interface {
		Handle(ctx context.Context, err error) []*Response
	}

	//Middleware represents intent handler interceptor/middleware
	Middleware func(Handler) Handler

	//Dispatcher dispatches intent to appropriate handler
	Dispatcher struct {
		//Middleware gives ability to add interceptor before intent handlers execution

		middlewares []Middleware
		initSync    sync.Once

		//Intents    map[string]Handler
		Handler    Handler
		ErrHandler ErrorHandler
		NLP        *nlp.IntentParser
	}

	//Request is parsed user question representation
	Request struct {
		Intent     string
		Raw        string
		Params     map[string]string
		Confidence float64
	}

	//Response is platform-agnostic answer representation
	Response struct {
		Text    string
		Buttons []*Button
	}

	//Button is platform-agnostic button representation
	Button struct {
		Text string
		Data string
	}

	// The HandlerFunc type is an adapter to allow the use of
	// ordinary functions as intent handlers.  If f is a function
	// with the appropriate signature, HandlerFunc(f) is a
	// Handler object that calls f.
	HandlerFunc func(ctx context.Context, rq *Request) ([]*Response, error)

	//ErrorHandlerFunc type is an adapter to allow the use of
	// ordinary functions as intent error handlers.
	ErrorHandlerFunc func(ctx context.Context, err error) []*Response
)

// Handle calls f(w, r).
func (f HandlerFunc) Handle(ctx context.Context, rq *Request) ([]*Response, error) {
	return f(ctx, rq)
}

// Handle calls f(w, r).
func (f ErrorHandlerFunc) Handle(ctx context.Context, err error) []*Response {
	return f(ctx, err)
}

//NewHandlerFunc factory method to have better autocomplete while creating HandlerFunc
func NewHandlerFunc(f func(ctx context.Context, rq *Request) ([]*Response, error)) HandlerFunc {
	return f
}

//DispatchRQ dispatches parsed user question to appropriate handler
func (d *Dispatcher) DispatchRQ(ctx context.Context, rq *Request) (rs []*Response) {
	d.init()

	defer func() {
		if r := recover(); r != nil {
			rs = d.ErrHandler.Handle(ctx, errors.Errorf("%s", r))
			return
		}
	}()

	var err error
	rs, err = d.Handler.Handle(ctx, rq)
	if nil != err {
		//some error occur. Convert to human-readable response
		rs = d.ErrHandler.Handle(ctx, err)

	}

	return
}

// Use adds IntentHandler middleware to dispatcher
// Middlewares are executed on each request in order they have been added
func (d *Dispatcher) Use(m func(Handler) Handler) *Dispatcher {
	d.middlewares = append(d.middlewares, m)
	return d
}

//Dispatch parses user question and then dispatches to appropriate handler
func (d *Dispatcher) Dispatch(ctx context.Context, msg string) (rs []*Response) {
	intent := d.NLP.Parse(msg)

	rq := &Request{
		Intent:     intent.Name,
		Params:     intent.Matches,
		Raw:        msg,
		Confidence: intent.Conf,
	}
	return d.DispatchRQ(ctx, rq)
}

//initializes dispatcher's internals
func (d *Dispatcher) init() {
	d.initSync.Do(func() {
		chain := chain(d.middlewares, d.Handler)
		if nil == chain {
			log.Fatal("Handler is not configured properly")
			//chain = HandlerFunc(d.ErrHandler.Handle)
		}
		d.Handler = chain
	})
}

// chain builds a Intent handler composed of an inline middleware stack and root
// handler in the order they are passed.
func chain(middlewares []Middleware, endpoint Handler) Handler {
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

//NewResponse creates new instance of bot response
func NewResponse() *Response {
	return &Response{}
}

//WithText adds simple text to the response and returns itself
func (rs *Response) WithText(t string) *Response {
	rs.Text = t
	return rs
}

//WithButtons adds buttons
func (rs *Response) WithButtons(btns ...*Button) *Response {
	rs.Buttons = btns
	return rs
}

//Respond collects multiple responses into the array
func Respond(rss ...*Response) []*Response {
	return rss
}
