package generator

// Options represents the options for the generator.
type Options struct {
	// Service is the name of the service the generator will generate files
	// for.
	Service string
	// Vendor is the service vendor.
	Vendor string
	// Directory is the directory where the files will be generated to.
	Directory string

	// Client determines whether or not the project is a client project.
	Client bool
	// Jaeger determines whether or not Jaeger integration is enabled.
	Jaeger bool
	// Jaeger determines whether or not Skaffold integration is enabled.
	Skaffold bool
}

// Option manipulates the Options passed.
type Option func(o *Options)

// Service sets the service name.
func Service(s string) Option {
	return func(o *Options) {
		o.Service = s
	}
}

// Vendor sets the service vendor.
func Vendor(v string) Option {
	return func(o *Options) {
		o.Vendor = v
	}
}

// Directory sets the directory in which files are generated.
func Directory(d string) Option {
	return func(o *Options) {
		o.Directory = d
	}
}

// Client sets whether or not the project is a client project.
func Client(c bool) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// Jaeger sets whether or not Jaeger integration is enabled.
func Jaeger(j bool) Option {
	return func(o *Options) {
		o.Jaeger = j
	}
}

// Skaffold sets whether or not Skaffold integration is enabled.
func Skaffold(s bool) Option {
	return func(o *Options) {
		o.Skaffold = s
	}
}
