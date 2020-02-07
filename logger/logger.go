// Package log provides a log interface
package logger

import (
	"fmt"
	"sync"
)

// Logger is a generic logging interface
type Logger interface {
	// Init initialises options
	Init(options ...Option) error
	// Level returns the logging level
	Level() Level
	// Log inserts a log entry.  Arguments may be handled in the manner
	// of fmt.Print, but the underlying logger may also decide to handle
	// them differently.
	Log(level Level, v ...interface{})
	// Logf insets a log entry.  Arguments are handled in the manner of
	// fmt.Printf.
	Logf(level Level, format string, v ...interface{})
	// Fields set fields to always be logged
	Fields(fields ...Field) Logger
	// SetLevel updates the logging level.
	SetLevel(Level)
	// String returns the name of logger
	String() string
}

var (
	mtx       sync.Mutex
	loggerMap = map[string]Logger{}
)

func Register(logger Logger) {
	mtx.Lock()
	defer mtx.Unlock()

	loggerMap[logger.String()] = logger
}

func GetLogger(name string) (Logger, error) {
	l := loggerMap[name]
	if l == nil {
		return nil, fmt.Errorf("no such name logger found %s", name)
	}

	return l, nil
}
