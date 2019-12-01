// Package log provides debug logging
package log

import (
	"fmt"
	"os"
	"time"
)

var (
	// DefaultLog logger
	DefaultLog = NewLog()
	// DefaultLevel is default log level
	DefaultLevel = LevelInfo
	// prefix for all messages
	prefix string
)

// Log is event log
type Log interface {
	// Read reads log entries from the logger
	Read(...ReadOption) []Record
	// Write writes records to log
	Write(Record)
	// Stream log records
	Stream(chan bool) <-chan Record
}

// Record is log record entry
type Record struct {
	// Timestamp of logged event
	Timestamp time.Time
	// Value contains log entry
	Value interface{}
	// Metadata to enrich log record
	Metadata map[string]string
}

// level is a log level
type Level int

const (
	LevelFatal Level = iota
	LevelError
	LevelInfo
	LevelWarn
	LevelDebug
	LevelTrace
)

func init() {
	switch os.Getenv("MICRO_LOG_LEVEL") {
	case "trace":
		DefaultLevel = LevelTrace
	case "debug":
		DefaultLevel = LevelDebug
	case "warn":
		DefaultLevel = LevelWarn
	case "info":
		DefaultLevel = LevelInfo
	case "error":
		DefaultLevel = LevelError
	case "fatal":
		DefaultLevel = LevelFatal
	}
}

func log(v ...interface{}) {
	if len(prefix) > 0 {
		DefaultLog.Write(Record{Value: fmt.Sprint(append([]interface{}{prefix, " "}, v...)...)})
		return
	}
	DefaultLog.Write(Record{Value: fmt.Sprint(v...)})
}

func logf(format string, v ...interface{}) {
	if len(prefix) > 0 {
		format = prefix + " " + format
	}
	DefaultLog.Write(Record{Value: fmt.Sprintf(format, v...)})
}

// WithLevel logs with the level specified
func WithLevel(l Level, v ...interface{}) {
	if l > DefaultLevel {
		return
	}
	log(v...)
}

// WithLevel logs with the level specified
func WithLevelf(l Level, format string, v ...interface{}) {
	if l > DefaultLevel {
		return
	}
	logf(format, v...)
}

// Trace provides trace level logging
func Trace(v ...interface{}) {
	WithLevel(LevelTrace, v...)
}

// Tracef provides trace level logging
func Tracef(format string, v ...interface{}) {
	WithLevelf(LevelTrace, format, v...)
}

// Debug provides debug level logging
func Debug(v ...interface{}) {
	WithLevel(LevelDebug, v...)
}

// Debugf provides debug level logging
func Debugf(format string, v ...interface{}) {
	WithLevelf(LevelDebug, format, v...)
}

// Warn provides warn level logging
func Warn(v ...interface{}) {
	WithLevel(LevelWarn, v...)
}

// Warnf provides warn level logging
func Warnf(format string, v ...interface{}) {
	WithLevelf(LevelWarn, format, v...)
}

// Info provides info level logging
func Info(v ...interface{}) {
	WithLevel(LevelInfo, v...)
}

// Infof provides info level logging
func Infof(format string, v ...interface{}) {
	WithLevelf(LevelInfo, format, v...)
}

// Error provides warn level logging
func Error(v ...interface{}) {
	WithLevel(LevelError, v...)
}

// Errorf provides warn level logging
func Errorf(format string, v ...interface{}) {
	WithLevelf(LevelError, format, v...)
}

// Fatal logs with Log and then exits with os.Exit(1)
func Fatal(v ...interface{}) {
	WithLevel(LevelFatal, v...)
	os.Exit(1)
}

// Fatalf logs with Logf and then exits with os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	WithLevelf(LevelFatal, format, v...)
	os.Exit(1)
}

// SetLevel sets the log level
func SetLevel(l Level) {
	DefaultLevel = l
}

// GetLevel returns the current level
func GetLevel() Level {
	return DefaultLevel
}

// Set a prefix for the logger
func SetPrefix(p string) {
	prefix = p
}

// Set service name
func Name(name string) {
	prefix = fmt.Sprintf("[%s]", name)
}
