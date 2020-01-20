package build

type Options struct {
	// local path to download source
	Path string
}

type Option func(o *Options)

// Local path for repository
func Path(p string) Option {
	return func(o *Options) {
		o.Path = p
	}
}
