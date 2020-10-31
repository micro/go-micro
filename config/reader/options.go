package reader

type Options struct {
	DisableReplaceEnvVars bool
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	return options
}

// WithDisableReplaceEnvVars disables the environment variable interpolation preprocessor
func WithDisableReplaceEnvVars() Option {
	return func(o *Options) {
		o.DisableReplaceEnvVars = true
	}
}
