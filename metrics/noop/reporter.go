package noop

import (
	"time"

	log "github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/metrics"
)

// Reporter is an implementation of metrics.Reporter:
type Reporter struct {
	options metrics.Options
}

// New returns a configured noop reporter:
func New(opts ...metrics.Option) *Reporter {
	log.Info("Metrics/NoOp - not doing anything")

	return &Reporter{
		options: metrics.NewOptions(opts...),
	}
}

// Count implements the metrics.Reporter interface Count method:
func (r *Reporter) Count(metricName string, value int64, tags metrics.Tags) error {
	return nil
}

// Gauge implements the metrics.Reporter interface Gauge method:
func (r *Reporter) Gauge(metricName string, value float64, tags metrics.Tags) error {
	return nil
}

// Timing implements the metrics.Reporter interface Timing method:
func (r *Reporter) Timing(metricName string, value time.Duration, tags metrics.Tags) error {
	return nil
}
