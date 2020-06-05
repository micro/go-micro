package resolver

type Options struct {
	Handler   string
	Namespace string
}

type Option func(o *Options)

// NewOptions returns new initialised options
func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if len(options.Namespace) == 0 {
		options.Namespace = "go.micro"
	}

	return options
}

// WithHandler sets the handler being used
func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

// WithNamespace sets the namespace for a request, e.g. go.micro.api
func WithNamespace(n string) Option {
	return func(o *Options) {
		o.Namespace = n
	}
}

// ResolveOptions are used when resolving a request
type ResolveOptions struct {
	Network string
}

// ResolveOption sets an option
type ResolveOption func(*ResolveOptions)

// WithNetwork sets the resolve network option
func WithNetwork(n string) ResolveOption {
	return func(o *ResolveOptions) {
		o.Network = n
	}
}

// NewResolveOptions returns new initialised resolve options
func NewResolveOptions(opts ...ResolveOption) ResolveOptions {
	var options ResolveOptions
	for _, o := range opts {
		o(&options)
	}

	if len(options.Network) == 0 {
		options.Network = "micro"
	}

	return options
}
