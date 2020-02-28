// Package log provides a log interface
package logger

var (
	// Default logger
	DefaultLogger Logger = NewHelper(NewLogger())
)

// Logger is a generic logging interface
type Logger interface {
	// Init initialises options
	Init(options ...Option) error
	// The Logger options
	Options() Options
	// Error set `error` field to be logged
	WithError(err error) Logger
	// Fields set fields to always be logged
	WithFields(fields map[string]interface{}) Logger
	// Log writes a log entry
	Log(level Level, v ...interface{})
	// Logf writes a formatted log entry
	Logf(level Level, format string, v ...interface{})
	// String returns the name of logger
	String() string
}

func Init(opts ...Option) error {
	return DefaultLogger.Init(opts...)
}
