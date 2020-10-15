package trace

type Options struct {
	// Size is the size of ring buffer
	Size int
}

type Option func(o *Options)

type ReadOptions struct {
	// Trace id
	Trace string
}

type ReadOption func(o *ReadOptions)

// Read the given trace
func ReadTrace(t string) ReadOption {
	return func(o *ReadOptions) {
		o.Trace = t
	}
}

const (
	// DefaultSize of the buffer
	DefaultSize = 64
)

// DefaultOptions returns default options
func DefaultOptions() Options {
	return Options{
		Size: DefaultSize,
	}
}
