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

// basic Logger is the default global
var globalLogger Logger = NewLogger()

func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

func SetGlobalLevel(lvl Level) {
	globalLogger.SetLevel(lvl)
}

func Info(args ...interface{}) {
	globalLogger.Log(InfoLevel, "", args, nil)
}
func Infof(template string, args ...interface{}) {
	globalLogger.Log(InfoLevel, template, args, nil)
}
func Infow(msg string, fields Fields) {
	globalLogger.Log(InfoLevel, msg, nil, fields)
}

func Trace(args ...interface{}) {
	globalLogger.Log(TraceLevel, "", args, nil)
}
func Tracef(template string, args ...interface{}) {
	globalLogger.Log(TraceLevel, template, args, nil)
}
func Tracew(msg string, fields Fields) {
	globalLogger.Log(TraceLevel, msg, nil, fields)
}

func Debug(args ...interface{}) {
	globalLogger.Log(DebugLevel, "", args, nil)
}
func Debugf(template string, args ...interface{}) {
	globalLogger.Log(DebugLevel, template, args, nil)
}
func Debugw(msg string, fields Fields) {
	globalLogger.Log(DebugLevel, msg, nil, fields)
}

func Warn(args ...interface{}) {
	globalLogger.Log(WarnLevel, "", args, nil)
}
func Warnf(template string, args ...interface{}) {
	globalLogger.Log(WarnLevel, template, args, nil)
}
func Warnw(msg string, fields Fields) {
	globalLogger.Log(WarnLevel, msg, nil, fields)
}

func Error(args ...interface{}) {
	globalLogger.Log(ErrorLevel, "", args, nil)
}
func Errorf(template string, args ...interface{}) {
	globalLogger.Log(ErrorLevel, template, args, nil)
}
func Errorw(msg string, err error) {
	globalLogger.Error(ErrorLevel, msg, nil, err)
}

func Panic(args ...interface{}) {
	globalLogger.Log(PanicLevel, "", args, nil)
}
func Panicf(template string, args ...interface{}) {
	globalLogger.Log(PanicLevel, template, args, nil)
}
func Panicw(msg string, fields Fields) {
	globalLogger.Log(PanicLevel, msg, nil, fields)
}

func Fatal(args ...interface{}) {
	globalLogger.Log(FatalLevel, "", args, nil)
}
func Fatalf(template string, args ...interface{}) {
	globalLogger.Log(FatalLevel, template, args, nil)
}
func Fatalw(msg string, fields Fields) {
	globalLogger.Log(FatalLevel, msg, nil, fields)
}
