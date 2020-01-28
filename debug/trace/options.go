package trace

type Options struct{}

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
