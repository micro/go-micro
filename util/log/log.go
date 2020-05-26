// Package log is a global internal logger
// DEPRECATED: this is frozen package, use github.com/micro/go-micro/v2/logger
package log

import (
	"fmt"
	"os"
	"sync/atomic"

	dlog "github.com/micro/go-micro/v2/debug/log"
	nlog "github.com/micro/go-micro/v2/logger"
)

// level is a log level
type Level int32

const (
	LevelFatal Level = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

type elog struct {
	dlog dlog.Log
}

var (
	// the local logger
	logger dlog.Log = &elog{}

	// default log level is info
	level = LevelInfo

	// prefix for all messages
	prefix string
)

func levelToLevel(l Level) nlog.Level {
	switch l {
	case LevelTrace:
		return nlog.TraceLevel
	case LevelDebug:
		return nlog.DebugLevel
	case LevelWarn:
		return nlog.WarnLevel
	case LevelInfo:
		return nlog.InfoLevel
	case LevelError:
		return nlog.ErrorLevel
	case LevelFatal:
		return nlog.FatalLevel
	}
	return nlog.InfoLevel
}

func init() {
	switch os.Getenv("MICRO_LOG_LEVEL") {
	case "trace":
		level = LevelTrace
	case "debug":
		level = LevelDebug
	case "warn":
		level = LevelWarn
	case "info":
		level = LevelInfo
	case "error":
		level = LevelError
	case "fatal":
		level = LevelFatal
	}
}

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelWarn:
		return "warn"
	case LevelInfo:
		return "info"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

func (el *elog) Read(opt ...dlog.ReadOption) ([]dlog.Record, error) {
	return el.dlog.Read(opt...)
}

func (el *elog) Write(r dlog.Record) error {
	return el.dlog.Write(r)
}

func (el *elog) Stream() (dlog.Stream, error) {
	return el.dlog.Stream()
}

// Log makes use of github.com/micro/debug/log
func Log(v ...interface{}) {
	if len(prefix) > 0 {
		v = append([]interface{}{prefix, " "}, v...)
	}
	nlog.DefaultLogger.Log(levelToLevel(level), v)
}

// Logf makes use of github.com/micro/debug/log
func Logf(format string, v ...interface{}) {
	if len(prefix) > 0 {
		format = prefix + " " + format
	}
	nlog.DefaultLogger.Log(levelToLevel(level), format, v)
}

// WithLevel logs with the level specified
func WithLevel(l Level, v ...interface{}) {
	if l > level {
		return
	}
	Log(v...)
}

// WithLevel logs with the level specified
func WithLevelf(l Level, format string, v ...interface{}) {
	if l > level {
		return
	}
	Logf(format, v...)
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
}

// Fatalf logs with Logf and then exits with os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	WithLevelf(LevelFatal, format, v...)
}

// SetLogger sets the local logger
func SetLogger(l dlog.Log) {
	logger = l
}

// GetLogger returns the local logger
func GetLogger() dlog.Log {
	return logger
}

// SetLevel sets the log level
func SetLevel(l Level) {
	atomic.StoreInt32((*int32)(&level), int32(l))
}

// GetLevel returns the current level
func GetLevel() Level {
	return level
}

// Set a prefix for the logger
func SetPrefix(p string) {
	prefix = p
}

// Set service name
func Name(name string) {
	prefix = fmt.Sprintf("[%s]", name)
}
