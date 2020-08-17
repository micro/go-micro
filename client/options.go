package client

import (
	"context"
	"time"

	"github.com/micro/go-micro/v3/broker"
	"github.com/micro/go-micro/v3/broker/http"
	"github.com/micro/go-micro/v3/codec"
	"github.com/micro/go-micro/v3/registry"
	"github.com/micro/go-micro/v3/router"
	regRouter "github.com/micro/go-micro/v3/router/registry"
	"github.com/micro/go-micro/v3/selector"
	"github.com/micro/go-micro/v3/selector/random"
	"github.com/micro/go-micro/v3/transport"
	thttp "github.com/micro/go-micro/v3/transport/http"
)

type Options struct {
	// Used to select codec
	ContentType string
	// Proxy address to send requests via
	Proxy string

	// Plugged interfaces
	Broker    broker.Broker
	Codecs    map[string]codec.NewCodec
	Router    router.Router
	Selector  selector.Selector
	Transport transport.Transport

	// Lookup used for looking up routes
	Lookup LookupFunc

	// Connection Pool
	PoolSize int
	PoolTTL  time.Duration

	// Response cache
	Cache *Cache

	// Middleware for client
	Wrappers []Wrapper

	// Default Call Options
	CallOptions CallOptions

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type CallOptions struct {
	// Address of remote hosts
	Address []string
	// Backoff func
	Backoff BackoffFunc
	// Duration to cache the response for
	CacheExpiry time.Duration
	// Transport Dial Timeout
	DialTimeout time.Duration
	// Number of Call attempts
	Retries int
	// Check if retriable func
	Retry RetryFunc
	// Request/Response timeout
	RequestTimeout time.Duration
	// Router to use for this call
	Router router.Router
	// Selector to use for the call
	Selector selector.Selector
	// SelectOptions to use when selecting a route
	SelectOptions []selector.SelectOption
	// Stream timeout for the stream
	StreamTimeout time.Duration
	// Use the auth token as the authorization header
	AuthToken bool
	// Network to lookup the route within
	Network string

	// Middleware for low level call func
	CallWrappers []CallWrapper

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type PublishOptions struct {
	// Exchange is the routing exchange for the message
	Exchange string
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type MessageOptions struct {
	ContentType string
}

type RequestOptions struct {
	ContentType string
	Stream      bool

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func NewOptions(options ...Option) Options {
	opts := Options{
		Cache:       NewCache(),
		Context:     context.Background(),
		ContentType: "application/protobuf",
		Codecs:      make(map[string]codec.NewCodec),
		CallOptions: CallOptions{
			Backoff:        DefaultBackoff,
			Retry:          DefaultRetry,
			Retries:        DefaultRetries,
			RequestTimeout: DefaultRequestTimeout,
			DialTimeout:    transport.DefaultDialTimeout,
		},
		Lookup:    LookupRoute,
		PoolSize:  DefaultPoolSize,
		PoolTTL:   DefaultPoolTTL,
		Broker:    http.NewBroker(),
		Router:    regRouter.NewRouter(),
		Selector:  random.NewSelector(),
		Transport: thttp.NewTransport(),
	}

	for _, o := range options {
		o(&opts)
	}

	return opts
}

// Broker to be used for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs[contentType] = c
	}
}

// Default content type of the client
func ContentType(ct string) Option {
	return func(o *Options) {
		o.ContentType = ct
	}
}

// Proxy sets the proxy address
func Proxy(addr string) Option {
	return func(o *Options) {
		o.Proxy = addr
	}
}

// PoolSize sets the connection pool size
func PoolSize(d int) Option {
	return func(o *Options) {
		o.PoolSize = d
	}
}

// PoolTTL sets the connection pool ttl
func PoolTTL(d time.Duration) Option {
	return func(o *Options) {
		o.PoolTTL = d
	}
}

// Transport to use for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// Registry sets the routers registry
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Router.Init(router.Registry(r))
	}
}

// Router is used to lookup routes for a service
func Router(r router.Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

// Selector is used to select a route
func Selector(s selector.Selector) Option {
	return func(o *Options) {
		o.Selector = s
	}
}

// Adds a Wrapper to a list of options passed into the client
func Wrap(w Wrapper) Option {
	return func(o *Options) {
		o.Wrappers = append(o.Wrappers, w)
	}
}

// Adds a Wrapper to the list of CallFunc wrappers
func WrapCall(cw ...CallWrapper) Option {
	return func(o *Options) {
		o.CallOptions.CallWrappers = append(o.CallOptions.CallWrappers, cw...)
	}
}

// Backoff is used to set the backoff function used
// when retrying Calls
func Backoff(fn BackoffFunc) Option {
	return func(o *Options) {
		o.CallOptions.Backoff = fn
	}
}

// Lookup sets the lookup function to use for resolving service names
func Lookup(l LookupFunc) Option {
	return func(o *Options) {
		o.Lookup = l
	}
}

// Number of retries when making the request.
// Should this be a Call Option?
func Retries(i int) Option {
	return func(o *Options) {
		o.CallOptions.Retries = i
	}
}

// Retry sets the retry function to be used when re-trying.
func Retry(fn RetryFunc) Option {
	return func(o *Options) {
		o.CallOptions.Retry = fn
	}
}

// The request timeout.
// Should this be a Call Option?
func RequestTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.RequestTimeout = d
	}
}

// StreamTimeout sets the stream timeout
func StreamTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.StreamTimeout = d
	}
}

// Transport dial timeout
func DialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.DialTimeout = d
	}
}

// Call Options

// WithExchange sets the exchange to route a message through
func WithExchange(e string) PublishOption {
	return func(o *PublishOptions) {
		o.Exchange = e
	}
}

// PublishContext sets the context in publish options
func PublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
	}
}

// WithAddress sets the remote addresses to use rather than using service discovery
func WithAddress(a ...string) CallOption {
	return func(o *CallOptions) {
		o.Address = a
	}
}

// WithCallWrapper is a CallOption which adds to the existing CallFunc wrappers
func WithCallWrapper(cw ...CallWrapper) CallOption {
	return func(o *CallOptions) {
		o.CallWrappers = append(o.CallWrappers, cw...)
	}
}

// WithBackoff is a CallOption which overrides that which
// set in Options.CallOptions
func WithBackoff(fn BackoffFunc) CallOption {
	return func(o *CallOptions) {
		o.Backoff = fn
	}
}

// WithRetry is a CallOption which overrides that which
// set in Options.CallOptions
func WithRetry(fn RetryFunc) CallOption {
	return func(o *CallOptions) {
		o.Retry = fn
	}
}

// WithRetries is a CallOption which overrides that which
// set in Options.CallOptions
func WithRetries(i int) CallOption {
	return func(o *CallOptions) {
		o.Retries = i
	}
}

// WithRequestTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithRequestTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.RequestTimeout = d
	}
}

// WithStreamTimeout sets the stream timeout
func WithStreamTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.StreamTimeout = d
	}
}

// WithDialTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithDialTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.DialTimeout = d
	}
}

// WithAuthToken is a CallOption which overrides the
// authorization header with the services own auth token
func WithAuthToken() CallOption {
	return func(o *CallOptions) {
		o.AuthToken = true
	}
}

// WithCache is a CallOption which sets the duration the response
// shoull be cached for
func WithCache(c time.Duration) CallOption {
	return func(o *CallOptions) {
		o.CacheExpiry = c
	}
}

// WithNetwork is a CallOption which sets the network attribute
func WithNetwork(n string) CallOption {
	return func(o *CallOptions) {
		o.Network = n
	}
}

// WithRouter sets the router to use for this call
func WithRouter(r router.Router) CallOption {
	return func(o *CallOptions) {
		o.Router = r
	}
}

// WithSelector sets the selector to use for this call
func WithSelector(s selector.Selector) CallOption {
	return func(o *CallOptions) {
		o.Selector = s
	}
}

// WithSelectOptions sets the options to pass to the selector for this call
func WithSelectOptions(sops ...selector.SelectOption) CallOption {
	return func(o *CallOptions) {
		o.SelectOptions = sops
	}
}

func WithMessageContentType(ct string) MessageOption {
	return func(o *MessageOptions) {
		o.ContentType = ct
	}
}

// Request Options

func WithContentType(ct string) RequestOption {
	return func(o *RequestOptions) {
		o.ContentType = ct
	}
}

func StreamingRequest() RequestOption {
	return func(o *RequestOptions) {
		o.Stream = true
	}
}
