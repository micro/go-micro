package stats

import (
	"runtime"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/util/ring"
)

type stats struct {
	// used to store past stats
	buffer *ring.Buffer

	sync.RWMutex
	started  int64
	requests uint64
	errors   uint64
}

func (s *stats) snapshot() *Stat {
	s.RLock()
	defer s.RUnlock()

	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	now := time.Now().Unix()

	return &Stat{
		Timestamp: now,
		Started:   s.started,
		Uptime:    now - s.started,
		Memory:    mstat.Alloc,
		GC:        mstat.PauseTotalNs,
		Threads:   uint64(runtime.NumGoroutine()),
		Requests:  s.requests,
		Errors:    s.errors,
	}
}

func (s *stats) Read() ([]*Stat, error) {
	// TODO adjustable size and optional read values
	buf := s.buffer.Get(60)
	var stats []*Stat

	// get a value from the buffer if it exists
	for _, b := range buf {
		stat, ok := b.Value.(*Stat)
		if !ok {
			continue
		}
		stats = append(stats, stat)
	}

	// get a snapshot
	stats = append(stats, s.snapshot())

	return stats, nil
}

func (s *stats) Write(stat *Stat) error {
	s.buffer.Put(stat)
	return nil
}

func (s *stats) Record(err error) error {
	s.Lock()
	defer s.Unlock()

	// increment the total request count
	s.requests++

	// increment the error count
	if err != nil {
		s.errors++
	}

	return nil
}

// NewStats returns a new in memory stats buffer
// TODO add options
func NewStats() Stats {
	return &stats{
		started: time.Now().Unix(),
		buffer:  ring.New(60),
	}
}
