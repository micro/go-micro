// Package stats provides process statistics
package stats

// Stats provides metrics recording and retrieval
type Stats interface {
	Read(...ReadOption) []*Metrics
	Record(*Metrics) error
	String() string
}

type Metrics struct {
	// Unique id of metric
	Id string
	// Metadata
	Metadata map[string]string
	// Floating values
	Values map[string]float64
	// Counters
	Counters map[string]int64
}

type ReadOptions struct{}

type ReadOption func(o *ReadOptions)
