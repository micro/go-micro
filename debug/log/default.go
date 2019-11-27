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

// defaultLogger is default micro logger
type defaultLogger struct {
	*buffer.Buffer
}

// NewLogger returns default Logger with
func NewLogger(opts ...Option) Logger {
	// get default options
	options := DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &defaultLogger{
		Buffer: buffer.New(options.Size),
	}
}

// Write writes log into logger
func (l *defaultLogger) Write(v ...interface{}) {
	l.log(fmt.Sprint(v...))
	golog.Print(v...)
}

// Read reads logs from the logger
func (l *defaultLogger) Read(n int) []interface{} {
	return l.Get(n)
}

func (l *defaultLogger) log(entry string) {
	l.Buffer.Put(entry)
}
