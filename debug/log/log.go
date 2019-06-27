// Package log provides a logger
package log

// A logging interface
type Log interface {
	// Write to the log
	Write(v ...interface{}) error
	// Read the log
	Read() ([]*Message, error)
}

// A log message
type Message struct {
	// Unique ID of the message
	Id string
	// Unix Timestamp
	Timestamp int64
	// Header of the log
	Header map[string]string
	// Associated data
	Body []byte
}
