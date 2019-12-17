package stats

import (
	"github.com/micro/go-micro/util/ring"
)

type stats struct {
	buffer *ring.Buffer
}

func (s *stats) Read() ([]*Stat, error) {
	// TODO adjustable size and optional read values
	buf := s.buffer.Get(1)
	var stats []*Stat

	for _, b := range buf {
		stat, ok := b.Value.(*Stat)
		if !ok {
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (s *stats) Write(stat *Stat) error {
	s.buffer.Put(stat)
	return nil
}

// NewStats returns a new in memory stats buffer
// TODO add options
func NewStats() Stats {
	return &stats{
		buffer: ring.New(1024),
	}
}
