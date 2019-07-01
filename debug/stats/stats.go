// Package stats provides metrics
package stats

import (
	"time"
)

type Stats interface {
	// Get the stats
	Get(id string) (*Metric, error)
	// Record a stat
	Record(id string, m *Metric) error
	// History of metric
	History(id string) ([]*Metric, error)
}

// A single stat
type Metric struct {
	// Id of the metric
	Id string
	// Time of recording
	Timestamp time.Time
	// the metric values
	Values map[string]interface{}
}
