package model

// Options holds configuration for a Model.
type Options struct {
	// DSN is the data source name / connection string.
	DSN string
}

// WithDSN sets the data source name.
func WithDSN(dsn string) Option {
	return func(o *Options) {
		o.DSN = dsn
	}
}

// NewOptions creates Options with defaults applied.
func NewOptions(opts ...Option) Options {
	o := Options{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithTable overrides the auto-derived table name.
func WithTable(name string) RegisterOption {
	return func(s *Schema) {
		s.Table = name
	}
}
