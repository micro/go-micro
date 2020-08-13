package metrics

import "time"

// Tags is a map of fields to add to a metric:
type Tags map[string]string

// Reporter is the standard metrics interface:
type Reporter interface {
	Count(metricName string, value int64, tags Tags) error
	Gauge(metricName string, value float64, tags Tags) error
	Timing(metricName string, value time.Duration, tags Tags) error
}
