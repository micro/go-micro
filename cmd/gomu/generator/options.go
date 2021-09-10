package generator

type Options struct {
	Service   string
	Vendor    string
	Directory string

	Client   bool
	Jaeger   bool
	Skaffold bool
}

type Option func(o *Options)

func Service(s string) Option {
	return func(o *Options) {
		o.Service = s
	}
}

func Vendor(v string) Option {
	return func(o *Options) {
		o.Vendor = v
	}
}

func Directory(d string) Option {
	return func(o *Options) {
		o.Directory = d
	}
}

func Client(c bool) Option {
	return func(o *Options) {
		o.Client = c
	}
}

func Jaeger(j bool) Option {
	return func(o *Options) {
		o.Jaeger = j
	}
}

func Skaffold(s bool) Option {
	return func(o *Options) {
		o.Skaffold = s
	}
}
