package prometheus

import (
	"errors"
	"time"

	"github.com/micro/go-micro/v3/metrics"
)

// ErrPrometheusPanic is a catch-all for the panics which can be thrown by the Prometheus client:
var ErrPrometheusPanic = errors.New("The Prometheus client panicked. Did you do something like change the tag cardinality or the type of a metric?")

// Count is a counter with key/value tags:
// New values are added to any previous one (eg "number of hits")
func (r *Reporter) Count(name string, value int64, tags metrics.Tags) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrometheusPanic
		}
	}()

	counter := r.metrics.getCounter(r.stripUnsupportedCharacters(name), r.listTagKeys(tags))
	metric, err := counter.GetMetricWith(r.convertTags(tags))
	if err != nil {
		return err
	}

	metric.Add(float64(value))
	return err
}

// Gauge is a register with key/value tags:
// New values simply override any previous one (eg "current connections")
func (r *Reporter) Gauge(name string, value float64, tags metrics.Tags) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrometheusPanic
		}
	}()

	gauge := r.metrics.getGauge(r.stripUnsupportedCharacters(name), r.listTagKeys(tags))
	metric, err := gauge.GetMetricWith(r.convertTags(tags))
	if err != nil {
		return err
	}

	metric.Set(value)
	return err
}

// Timing is a histogram with key/valye tags:
// New values are added into a series of aggregations
func (r *Reporter) Timing(name string, value time.Duration, tags metrics.Tags) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrometheusPanic
		}
	}()

	timing := r.metrics.getTiming(r.stripUnsupportedCharacters(name), r.listTagKeys(tags))
	metric, err := timing.GetMetricWith(r.convertTags(tags))
	if err != nil {
		return err
	}

	metric.Observe(value.Seconds())
	return err
}
