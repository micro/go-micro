package builder

// Options to use when building source
type Options struct {
	// Archive used, e.g. tar
	Archive string
	// Entrypoint to use, e.g. foo/main.go
	Entrypoint string
	// Env vars to pass to the builder
	Env []string
}

// Option configures one or more options
type Option func(o *Options)

// Archive sets the builders archive
func Archive(a string) Option {
	return func(o *Options) {
		o.Archive = a
	}
}

// Entrypoint sets the builders entrypoint
func Entrypoint(e string) Option {
	return func(o *Options) {
		o.Entrypoint = e
	}
}

// Env vars to pass to the builder
func Env(vars ...string) Option {
	return func(o *Options) {
		o.Env = vars
	}
}
