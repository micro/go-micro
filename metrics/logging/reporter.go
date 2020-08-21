package logging

import (
	"time"

	log "github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/metrics"
)

// Reporter is an implementation of metrics.Reporter:
type Reporter struct {
	logger  log.Logger
	options metrics.Options
}

// New returns a configured noop reporter:
func New(opts ...metrics.Option) *Reporter {
	options := metrics.NewOptions(opts...)
	logger := log.NewLogger(log.WithFields(convertTags(options.DefaultTags)))
	logger.Log(log.InfoLevel, "Metrics/Logging - metrics will be logged (at TRACE level)")

	return &Reporter{
		logger:  logger,
		options: metrics.NewOptions(opts...),
	}
}

// Count implements the metrics.Reporter interface Count method:
func (r *Reporter) Count(metricName string, value int64, tags metrics.Tags) error {
	r.logger.Logf(log.TraceLevel, "Count metric: %s", tags)
	return nil
}

// Gauge implements the metrics.Reporter interface Gauge method:
func (r *Reporter) Gauge(metricName string, value float64, tags metrics.Tags) error {
	r.logger.Logf(log.TraceLevel, "Gauge metric: %s", tags)
	return nil
}

// Timing implements the metrics.Reporter interface Timing method:
func (r *Reporter) Timing(metricName string, value time.Duration, tags metrics.Tags) error {
	r.logger.Logf(log.TraceLevel, "Timing metric: %s", tags)
	return nil
}

// convertTags turns Tags into prometheus labels:
func convertTags(tags metrics.Tags) map[string]interface{} {
	labels := make(map[string]interface{})
	for key, value := range tags {
		labels[key] = value
	}
	return labels
}
