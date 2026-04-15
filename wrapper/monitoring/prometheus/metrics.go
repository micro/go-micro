package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// metrics bundles the counters/histograms used by the wrappers.
// A metrics value is keyed by (name + namespace + subsystem) so that
// multiple wrappers created with the same options share the same
// collectors instead of failing with AlreadyRegisteredError.
type metrics struct {
	requestTotal    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// metricLabels are the labels we use on every metric. They intentionally
// stay on a small, low-cardinality set: high-cardinality labels (e.g. the
// full request body) must not end up in Prometheus.
var metricLabels = []string{"service", "endpoint", "status"}

var (
	metricsMu    sync.Mutex
	metricsCache = map[string]*metrics{}
)

// getMetrics returns a cached metrics bundle for the given options, creating
// and registering it on first use. Collectors that were already registered
// on the underlying Registerer (e.g. because a user constructed two wrappers
// with identical options) are reused transparently.
func getMetrics(opts Options) *metrics {
	key := opts.Name + "\x00" + opts.Namespace + "\x00" + opts.Subsystem

	metricsMu.Lock()
	defer metricsMu.Unlock()

	if m, ok := metricsCache[key]; ok {
		return m
	}

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   opts.Namespace,
			Subsystem:   opts.Subsystem,
			Name:        opts.Name + "_request_total",
			Help:        "How many go-micro requests processed, partitioned by service, endpoint and status.",
			ConstLabels: opts.ConstLabels,
		},
		metricLabels,
	)

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   opts.Namespace,
			Subsystem:   opts.Subsystem,
			Name:        opts.Name + "_request_duration_seconds",
			Help:        "Histogram of go-micro request latencies in seconds, partitioned by service, endpoint and status.",
			ConstLabels: opts.ConstLabels,
			Buckets:     opts.Buckets,
		},
		metricLabels,
	)

	m := &metrics{
		requestTotal:    register(opts.Registerer, counter).(*prometheus.CounterVec),
		requestDuration: register(opts.Registerer, histogram).(*prometheus.HistogramVec),
	}
	metricsCache[key] = m
	return m
}

// register registers c on r. If an identical collector is already registered
// (AlreadyRegisteredError), the existing collector is returned so that the
// wrapper can be constructed more than once without panicking.
func register(r prometheus.Registerer, c prometheus.Collector) prometheus.Collector {
	if err := r.Register(c); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector
		}
		// Any other registration error is a programming mistake (e.g.
		// inconsistent label dimensions) and should surface loudly.
		panic(err)
	}
	return c
}

// status returns "success" or "fail" depending on whether err is nil.
// Using a fixed, low-cardinality set keeps Prometheus memory bounded.
func status(err error) string {
	if err != nil {
		return "fail"
	}
	return "success"
}
