package trace

type Options struct{}

type Option func(o *Options)

type ReadOptions struct {
	// Trace id
	Trace string
}

type ReadOption func(o *ReadOptions)
