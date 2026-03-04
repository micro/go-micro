package model

// Option configures a Database.
type Option func(*DatabaseOptions)

// DatabaseOptions holds configuration for a Database backend.
type DatabaseOptions struct {
	// DSN is the data source name / connection string.
	DSN string
}

// WithDSN sets the data source name for the database connection.
func WithDSN(dsn string) Option {
	return func(o *DatabaseOptions) {
		o.DSN = dsn
	}
}

// NewDatabaseOptions creates DatabaseOptions with defaults applied.
func NewDatabaseOptions(opts ...Option) DatabaseOptions {
	o := DatabaseOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// ModelOption configures a Model instance.
type ModelOption func(*Schema)

// WithTable overrides the auto-derived table name.
func WithTable(name string) ModelOption {
	return func(s *Schema) {
		s.Table = name
	}
}
