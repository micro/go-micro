package metrics

var (
	// The Prometheus metrics will be made available on this port:
	defaultPrometheusListenAddress = ":9000"
	// This is the endpoint where the Prometheus metrics will be made available ("/metrics" is the default with Prometheus):
	defaultPath = "/metrics"
	// defaultPercentiles is the default spread of percentiles/quantiles we maintain for timings / histogram metrics:
	defaultPercentiles = []float64{0, 0.5, 0.75, 0.90, 0.95, 0.98, 0.99, 1}
)

// Option powers the configuration for metrics implementations:
type Option func(*Options)

// Options for metrics implementations:
type Options struct {
	Address     string
	DefaultTags Tags
	Path        string
	Percentiles []float64
}

// NewOptions prepares a set of options:
func NewOptions(opt ...Option) Options {
	opts := Options{
		Address:     defaultPrometheusListenAddress,
		DefaultTags: make(Tags),
		Path:        defaultPath,
		Percentiles: defaultPercentiles,
	}

	for _, o := range opt {
		o(&opts)
	}

	return opts
}

// Path used to serve metrics over HTTP:
func Path(value string) Option {
	return func(o *Options) {
		o.Path = value
	}
}

// Address is the listen address to serve metrics on:
func Address(value string) Option {
	return func(o *Options) {
		o.Address = value
	}
}

// DefaultTags will be added to every metric:
func DefaultTags(value Tags) Option {
	return func(o *Options) {
		o.DefaultTags = value
	}
}

// Percentiles defines the desired spread of statistics for histogram / timing metrics:
func Percentiles(value []float64) Option {
	return func(o *Options) {
		o.Percentiles = value
	}
}
