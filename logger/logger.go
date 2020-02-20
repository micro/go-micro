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
	// The Logger options
	Options() Options
	// Error set `error` field to be logged
	Error(err error) Logger
	// Fields set fields to always be logged
	Fields(fields map[string]interface{}) Logger
	// Log writes a log entry
	Log(level Level, v ...interface{})
	// Logf writes a formatted log entry
	Logf(level Level, format string, v ...interface{})
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

// GetLevel converts a level string into a logger Level value.
// returns an error if the input string does not match known values.
func GetLevel(levelStr string) (Level, error) {
	switch levelStr {
	case TraceLevel.String():
		return TraceLevel, nil
	case DebugLevel.String():
		return DebugLevel, nil
	case InfoLevel.String():
		return InfoLevel, nil
	case WarnLevel.String():
		return WarnLevel, nil
	case ErrorLevel.String():
		return ErrorLevel, nil
	case FatalLevel.String():
		return FatalLevel, nil
	case PanicLevel.String():
		return PanicLevel, nil
	}
	return InfoLevel, fmt.Errorf("Unknown Level String: '%s', defaulting to NoLevel", levelStr)
}
