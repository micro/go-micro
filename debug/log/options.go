package log

// Option used by the logger
type Option func(*Options)

// Options are logger options
type Options struct {
	// Size is the size of ring buffer
	Size int
}

// Size sets the size of the ring buffer
func Size(s int) Option {
	return func(o *Options) {
		o.Size = s
	}
}

// DefaultOptions returns default options
func DefaultOptions() Options {
	return Options{
		Size: DefaultSize,
	}
}
