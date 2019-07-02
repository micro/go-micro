// Package log is a global internal logger
package log

import (
	"os"

	"github.com/go-log/log"
	golog "github.com/go-log/log/log"
)

// level is a log level
type Level int

const (
	trace Level = iota
	debug
	info
	fatal
)

var (
	// the local logger
	logger log.Logger = golog.New()

	// default log level is debug
	level = info
)

// Log makes use of github.com/go-log/log.Log
func Log(v ...interface{}) {
	logger.Log(v...)
}

// Logf makes use of github.com/go-log/log.Logf
func Logf(format string, v ...interface{}) {
	logger.Logf(format, v...)
}

// WithLevel logs with the level specified
func WithLevel(l Level, v ...interface{}) {
	if l < level {
		return
	}
	Log(v...)
}

// WithLevel logs with the level specified
func WithLevelf(l Level, format string, v ...interface{}) {
	if l < level {
		return
	}
	Logf(format, v...)
}

// Trace provides trace level logging
func Trace(v ...interface{}) {
	WithLevel(trace, v...)
}

// Tracef provides trace level logging
func Tracef(format string, v ...interface{}) {
	WithLevelf(trace, format, v...)
}

// Debug provides debug level logging
func Debug(v ...interface{}) {
	WithLevel(debug, v...)
}

// Debugf provides debug level logging
func Debugf(format string, v ...interface{}) {
	WithLevelf(debug, format, v...)
}

// Info provides info level logging
func Info(v ...interface{}) {
	WithLevel(info, v...)
}

// Infof provides info level logging
func Infof(format string, v ...interface{}) {
	WithLevelf(info, format, v...)
}

// Fatal logs with Log and then exits with os.Exit(1)
func Fatal(v ...interface{}) {
	WithLevel(fatal, v...)
	os.Exit(1)
}

// Fatalf logs with Logf and then exits with os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	WithLevelf(fatal, format, v...)
	os.Exit(1)
}

// SetLogger sets the local logger
func SetLogger(l log.Logger) {
	logger = l
}

// GetLogger returns the local logger
func GetLogger() log.Logger {
	return logger
}

// SetLevel sets the log level
func SetLevel(l Level) {
	level = l
}

// GetLevel returns the current level
func GetLevel() Level {
	return level
}
