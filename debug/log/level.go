// Package log provides debug logging
package log

import (
	"os"
)

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
