package build

type Options struct {
	// Name for the package
	Name string
	// Version for the package
	Version string
	// Language of the source
	Language string
}

type Option func(o *Options)

// Name sets the name option
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Version sets the version option
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

// Language sets the language option
func Language(l string) Option {
	return func(o *Options) {
		o.Language = l
	}
}
