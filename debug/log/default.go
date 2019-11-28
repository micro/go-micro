package log

import (
	"fmt"
	golog "log"

	"github.com/micro/go-micro/debug/buffer"
)

var (
	// DefaultSize of the logger buffer
	DefaultSize = 1000
)

// defaultLog is default micro log
type defaultLog struct {
	*buffer.Buffer
}

// NewLog returns default Logger with
func NewLog(opts ...Option) Log {
	// get default options
	options := DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &defaultLog{
		Buffer: buffer.New(options.Size),
	}
}

// Write writes logs into logger
func (l *defaultLog) Write(v ...interface{}) {
	l.Buffer.Put(fmt.Sprint(v...))
	golog.Print(v...)
}

// Read reads logs and returns them
func (l *defaultLog) Read(opts ...ReadOption) []Record {
	options := ReadOptions{}
	// initialize the read options
	for _, o := range opts {
		o(&options)
	}

	var entries []*buffer.Entry
	// if Since options ha sbeen specified we honor it
	if !options.Since.IsZero() {
		entries = l.Buffer.Since(options.Since)
	}

	// only if we specified valid count constraint
	// do we do some serious if-else kung-fu
	// if since has been given we return *count* number of
	// logs since the requested timestamp;
	// otherwise we retourn last count number of logs
	if options.Count > 0 {
		switch len(entries) > 0 {
		case true:
			// if we request fewer logs than what since constraint gives us
			if options.Count < len(entries) {
				entries = entries[0:options.Count]
			}
		default:
			entries = l.Buffer.Get(options.Count)
		}
	}

	records := make([]Record, 0, len(entries))
	for _, entry := range entries {
		record := Record{
			Timestamp: entry.Timestamp,
			Value:     entry.Value,
		}
		records = append(records, record)
	}

	return records
}
