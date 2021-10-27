package jaeger

import (
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-lib/metrics"
)

var (
	// DefaultLogger is the default Jaeger logger.
	DefaultLogger = jaeger.StdLogger

	// DefaultMetrics is the default Jaeger metrics factory.
	DefaultMetrics = metrics.NullFactory
)

// Options represents the options passed to the Jaeger tracer.
type Options struct {
	Name         string
	FromEnv      bool
	GlobalTracer bool
	Logger       jaeger.Logger
	Metrics      metrics.Factory
}

// Option manipulates the passed Options struct.
type Option func(o *Options)

func newOptions(opts ...Option) Options {
	options := Options{
		Logger:  DefaultLogger,
		Metrics: DefaultMetrics,
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

// Name sets the service name for the Jaeger tracer.
func Name(s string) Option {
	return func(o *Options) {
		o.Name = s
	}
}

// FromEnv determines whether the Jaeger tracer configuration should use
// environment variables.
func FromEnv(e bool) Option {
	return func(o *Options) {
		o.FromEnv = e
	}
}

// GlobalTracer determines whether the Jaeger tracer should be set as the
// global tracer.
func GlobalTracer(e bool) Option {
	return func(o *Options) {
		o.GlobalTracer = e
	}
}

// Logger sets the logger for the Jaeger tracer.
func Logger(l jaeger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// Metrics sets the metrics factory for the Jaeger tracer.
func Metrics(m metrics.Factory) Option {
	return func(o *Options) {
		o.Metrics = m
	}
}
