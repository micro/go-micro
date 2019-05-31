// Package event provides a distributed log interface
package event

// Event provides a distributed log interface
type Event interface {
	// Log retrieves the log with an id/name
	Log(id string) (Log, error)
}

// Log is an individual event log
type Log interface {
	// Close the log handle
	Close() error
	// Log ID
	Id() string
	// Read will read the next record
	Read() (*Record, error)
	// Go to an offset
	Seek(offset int64) error
	// Write an event to the log
	Write(*Record) error
}

type Record struct {
	Metadata map[string]interface{}
	Data     []byte
}
