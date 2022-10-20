package client

import (
	"context"
	"time"

	"go-micro.dev/v4/broker"
	"go-micro.dev/v4/codec"
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/selector"
	"go-micro.dev/v4/transport"
)

var (
	// DefaultBackoff is the default backoff function for retries.
	DefaultBackoff = exponentialBackoff
	// DefaultRetry is the default check-for-retry function for retries.
	DefaultRetry = RetryOnError
	// DefaultRetries is the default number of times a request is tried.
	DefaultRetries = 5
	// DefaultRequestTimeout is the default request timeout.
	DefaultRequestTimeout = time.Second * 30
	// DefaultConnectionTimeout is the default connection timeout.
	DefaultConnectionTimeout = time.Second * 5
	// DefaultPoolSize sets the connection pool size.
	DefaultPoolSize = 100
	// DefaultPoolTTL sets the connection pool ttl.
	DefaultPoolTTL = time.Minute
)

// Options are the Client options.
type Options struct {
	// Used to select codec
	ContentType string

	// Plugged interfaces
	Broker    broker.Broker
	Codecs    map[string]codec.NewCodec
	Registry  registry.Registry
	Selector  selector.Selector
	Transport transport.Transport

	// Router sets the router
	Router Router

	// Connection Pool
	PoolSize int
	PoolTTL  time.Duration

	// Response cache
	Cache *Cache

	// Middleware for client
	Wrappers []Wrapper

	// Default Call Options
	CallOptions CallOptions

	// Logger is the underline logger
	Logger logger.Logger

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// CallOptions are options used to make calls to a server.
type CallOptions struct {
	SelectOptions []selector.SelectOption

	// Address of remote hosts
	Address []string
	// Backoff func
	Backoff BackoffFunc
	// Check if retriable func
	Retry RetryFunc
	// Number of Call attempts
	Retries int
	// Transport Dial Timeout. Used for initial dial to establish a connection.
	DialTimeout time.Duration
	// ConnectionTimeout of one request to the server.
	// Set this lower than the RequestTimeout to enbale retries on connection timeout.
	ConnectionTimeout time.Duration
	// Request/Response timeout of entire srv.Call, for single request timeout set ConnectionTimeout.
	RequestTimeout time.Duration
	// Stream timeout for the stream
	StreamTimeout time.Duration
	// Use the services own auth token
	ServiceToken bool
	// Duration to cache the response for
	CacheExpiry time.Duration
	// ConnClose sets the Connection: close header.
	ConnClose bool

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

// NewOptions creates new Client options.
func NewOptions(options ...Option) Options {
	opts := Options{
		Cache:       NewCache(),
		Context:     context.Background(),
		ContentType: DefaultContentType,
		Codecs:      make(map[string]codec.NewCodec),
		CallOptions: CallOptions{
			Backoff:           DefaultBackoff,
			Retry:             DefaultRetry,
			Retries:           DefaultRetries,
			RequestTimeout:    DefaultRequestTimeout,
			ConnectionTimeout: DefaultConnectionTimeout,
			DialTimeout:       transport.DefaultDialTimeout,
		},
		PoolSize:  DefaultPoolSize,
		PoolTTL:   DefaultPoolTTL,
		Broker:    broker.DefaultBroker,
		Selector:  selector.DefaultSelector,
		Registry:  registry.DefaultRegistry,
		Transport: transport.DefaultTransport,
		Logger:    logger.DefaultLogger,
	}

	for _, o := range options {
		o(&opts)
	}

	return opts
}

// Broker to be used for pub/sub.
func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

// Codec to be used to encode/decode requests for a given content type.
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs[contentType] = c
	}
}

// ContentType sets the default content type of the client.
func ContentType(ct string) Option {
	return func(o *Options) {
		o.ContentType = ct
	}
}

// PoolSize sets the connection pool size.
func PoolSize(d int) Option {
	return func(o *Options) {
		o.PoolSize = d
	}
}

// PoolTTL sets the connection pool ttl.
func PoolTTL(d time.Duration) Option {
	return func(o *Options) {
		o.PoolTTL = d
	}
}

// Registry to find nodes for a given service.
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
		// set in the selector
		o.Selector.Init(selector.Registry(r))
	}
}

// Transport to use for communication e.g http, rabbitmq, etc.
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// Select is used to select a node to route a request to.
func Selector(s selector.Selector) Option {
	return func(o *Options) {
		o.Selector = s
	}
}

// Adds a Wrapper to a list of options passed into the client.
func Wrap(w Wrapper) Option {
	return func(o *Options) {
		o.Wrappers = append(o.Wrappers, w)
	}
}

// Adds a Wrapper to the list of CallFunc wrappers.
func WrapCall(cw ...CallWrapper) Option {
	return func(o *Options) {
		o.CallOptions.CallWrappers = append(o.CallOptions.CallWrappers, cw...)
	}
}

// Backoff is used to set the backoff function used
// when retrying Calls.
func Backoff(fn BackoffFunc) Option {
	return func(o *Options) {
		o.CallOptions.Backoff = fn
	}
}

// Retries set the number of retries when making the request.
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

// RequestTimeout set the request timeout.
func RequestTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.RequestTimeout = d
	}
}

// StreamTimeout sets the stream timeout.
func StreamTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.StreamTimeout = d
	}
}

// DialTimeout sets the transport dial timeout.
func DialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.DialTimeout = d
	}
}

// Call Options

// WithExchange sets the exchange to route a message through.
func WithExchange(e string) PublishOption {
	return func(o *PublishOptions) {
		o.Exchange = e
	}
}

// PublishContext sets the context in publish options.
func PublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
	}
}

// WithAddress sets the remote addresses to use rather than using service discovery.
func WithAddress(a ...string) CallOption {
	return func(o *CallOptions) {
		o.Address = a
	}
}

func WithSelectOption(so ...selector.SelectOption) CallOption {
	return func(o *CallOptions) {
		o.SelectOptions = append(o.SelectOptions, so...)
	}
}

// WithCallWrapper is a CallOption which adds to the existing CallFunc wrappers.
func WithCallWrapper(cw ...CallWrapper) CallOption {
	return func(o *CallOptions) {
		o.CallWrappers = append(o.CallWrappers, cw...)
	}
}

// WithBackoff is a CallOption which overrides that which
// set in Options.CallOptions.
func WithBackoff(fn BackoffFunc) CallOption {
	return func(o *CallOptions) {
		o.Backoff = fn
	}
}

// WithRetry is a CallOption which overrides that which
// set in Options.CallOptions.
func WithRetry(fn RetryFunc) CallOption {
	return func(o *CallOptions) {
		o.Retry = fn
	}
}

// WithRetries sets the number of tries for a call.
// This CallOption overrides Options.CallOptions.
func WithRetries(i int) CallOption {
	return func(o *CallOptions) {
		o.Retries = i
	}
}

// WithRequestTimeout is a CallOption which overrides that which
// set in Options.CallOptions.
func WithRequestTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.RequestTimeout = d
	}
}

// WithConnClose sets the Connection header to close.
func WithConnClose() CallOption {
	return func(o *CallOptions) {
		o.ConnClose = true
	}
}

// WithStreamTimeout sets the stream timeout.
func WithStreamTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.StreamTimeout = d
	}
}

// WithDialTimeout is a CallOption which overrides that which
// set in Options.CallOptions.
func WithDialTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.DialTimeout = d
	}
}

// WithServiceToken is a CallOption which overrides the
// authorization header with the services own auth token.
func WithServiceToken() CallOption {
	return func(o *CallOptions) {
		o.ServiceToken = true
	}
}

// WithCache is a CallOption which sets the duration the response
// shoull be cached for.
func WithCache(c time.Duration) CallOption {
	return func(o *CallOptions) {
		o.CacheExpiry = c
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

// WithRouter sets the client router.
func WithRouter(r Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

// WithLogger sets the underline logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
