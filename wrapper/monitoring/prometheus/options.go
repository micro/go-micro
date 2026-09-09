// Package prometheus provides a go-micro wrapper that exposes standard
// request/response metrics (request count, latency, errors) to Prometheus.
package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Options holds configuration for the Prometheus wrapper.
type Options struct {
	// Name is the metric name prefix (Prometheus "name").
	// Default: "micro".
	Name string

	// Namespace for the Prometheus metrics.
	// Default: "" (empty).
	Namespace string

	// Subsystem for the Prometheus metrics.
	// Default: "" (empty).
	Subsystem string

	// ConstLabels are labels applied to every metric.
	ConstLabels prometheus.Labels

	// Buckets defines the histogram buckets (seconds) for latency.
	// When nil, prometheus.DefBuckets is used.
	Buckets []float64

	// Registerer is used to register the metrics.
	// When nil, prometheus.DefaultRegisterer is used.
	Registerer prometheus.Registerer
}

// Option applies a single configuration value.
type Option func(*Options)

// ServiceName sets the metric name prefix (Prometheus "name").
func ServiceName(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

// Namespace sets the Prometheus namespace for metrics.
func Namespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// Subsystem sets the Prometheus subsystem for metrics.
func Subsystem(subsystem string) Option {
	return func(o *Options) {
		o.Subsystem = subsystem
	}
}

// ConstLabels sets labels applied to every metric.
func ConstLabels(labels prometheus.Labels) Option {
	return func(o *Options) {
		o.ConstLabels = labels
	}
}

// Buckets sets the histogram buckets (in seconds) for latency metrics.
func Buckets(buckets []float64) Option {
	return func(o *Options) {
		o.Buckets = buckets
	}
}

// Registerer sets the Prometheus registerer used to register metrics.
// When unset, prometheus.DefaultRegisterer is used.
func Registerer(r prometheus.Registerer) Option {
	return func(o *Options) {
		o.Registerer = r
	}
}

// newOptions builds Options from the provided Option functions, applying
// sensible defaults.
func newOptions(opts ...Option) Options {
	options := Options{
		Name:       "micro",
		Buckets:    prometheus.DefBuckets,
		Registerer: prometheus.DefaultRegisterer,
	}
	for _, o := range opts {
		o(&options)
	}
	if options.Registerer == nil {
		options.Registerer = prometheus.DefaultRegisterer
	}
	if len(options.Buckets) == 0 {
		options.Buckets = prometheus.DefBuckets
	}
	return options
}
