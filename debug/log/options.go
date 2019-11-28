package log

import "time"

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

// ReadOptions for querying the logs
type ReadOptions struct {
	// Since what time in past to return the logs
	Since time.Time
	// Count specifies number of logs to return
	Count int
	// Stream requests continuous log stream
	Stream bool
}

// ReadOption used for reading the logs
type ReadOption func(*ReadOptions)

// Since sets the time since which to return the log records
func Since(s time.Time) ReadOption {
	return func(o *ReadOptions) {
		o.Since = s
	}
}

// Count sets the number of log records to return
func Count(c int) ReadOption {
	return func(o *ReadOptions) {
		o.Count = c
	}
}

// Stream requests continuous log stream
func Stream(s bool) ReadOption {
	return func(o *ReadOptions) {
		o.Stream = s
	}
}
