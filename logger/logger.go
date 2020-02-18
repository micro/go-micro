// Package log provides a log interface
package logger

import (
	"fmt"
)

type Fields map[string]interface{}

// Logger is a generic logging interface
type Logger interface {
	// Init initialises options
	Init(options ...Option) error
	// Level returns the logging level
	Level() Level
	// SetLevel updates the logging level.
	SetLevel(Level)
	// String returns the name of logger
	String() string
	// log at given level with message, fmtArgs and context fields
	Log(level Level, template string, fmtArgs []interface{}, fields Fields)
	// log error at given level with message, fmtArgs and stack if enabled.
	Error(level Level, template string, fmtArgs []interface{}, err error)
}

// ParseLevel converts a level string into a logger Level value.
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
