package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

func init() {
	lvl, err := GetLevel(os.Getenv("MICRO_LOG_LEVEL"))
	if err != nil {
		lvl = InfoLevel
	}

	DefaultLogger = NewLogger(WithLevel(lvl))
}

type defaultLogger struct {
	opts Options
	slog *slog.Logger
	sync.RWMutex
}

// Init (opts...) should only overwrite provided options.
func (l *defaultLogger) Init(opts ...Option) error {
	l.Lock()
	defer l.Unlock()

	for _, o := range opts {
		o(&l.opts)
	}

	// Recreate slog logger with new options
	handlerOpts := &slog.HandlerOptions{
		Level:     l.opts.Level.ToSlog(),
		AddSource: true,
	}

	// Create text handler for stdout
	textHandler := slog.NewTextHandler(l.opts.Out, handlerOpts)
	
	// Create debug log handler for debug/log buffer
	debugHandler := newDebugLogHandler(handlerOpts.Level)
	
	// Combine both handlers
	handler := newMultiHandler(textHandler, debugHandler)
	
	l.slog = slog.New(handler)

	// Add fields if any
	if len(l.opts.Fields) > 0 {
		const fieldsPerKV = 2
		args := make([]any, 0, len(l.opts.Fields)*fieldsPerKV)
		for k, v := range l.opts.Fields {
			args = append(args, k, v)
		}

		l.slog = l.slog.With(args...)
	}

	return nil
}

func (l *defaultLogger) String() string {
	return "default"
}

func (l *defaultLogger) Fields(fields map[string]interface{}) Logger {
	l.RLock()
	nfields := copyFields(l.opts.Fields)
	opts := l.opts
	l.RUnlock()

	for k, v := range fields {
		nfields[k] = v
	}

	// Create new logger without locks
	newLogger := NewLogger(
		WithLevel(opts.Level),
		WithFields(nfields),
		WithOutput(opts.Out),
		WithCallerSkipCount(opts.CallerSkipCount),
	)

	return newLogger
}

func copyFields(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}

func (l *defaultLogger) Log(level Level, v ...interface{}) {
	// TODO decide does we need to write message if log level not used?
	if !l.opts.Level.Enabled(level) {
		return
	}

	l.RLock()
	slogger := l.slog

	if slogger == nil {
		// Fallback if not initialized
		slogger = slog.Default()
	}
	l.RUnlock()

	// Get caller information
	var pcs [1]uintptr

	runtime.Callers(l.opts.CallerSkipCount, pcs[:])
	r := slog.NewRecord(time.Now(), level.ToSlog(), fmt.Sprint(v...), pcs[0])

	_ = slogger.Handler().Handle(context.Background(), r)
}

func (l *defaultLogger) Logf(level Level, format string, v ...interface{}) {
	//	 TODO decide does we need to write message if log level not used?
	if !l.opts.Level.Enabled(level) {
		return
	}

	l.RLock()
	slogger := l.slog

	if slogger == nil {
		// Fallback if not initialized
		slogger = slog.Default()
	}
	l.RUnlock()

	// Get caller information
	var pcs [1]uintptr

	runtime.Callers(l.opts.CallerSkipCount, pcs[:])
	r := slog.NewRecord(time.Now(), level.ToSlog(), fmt.Sprintf(format, v...), pcs[0])

	_ = slogger.Handler().Handle(context.Background(), r)
}

func (l *defaultLogger) Options() Options {
	// not guard against options Context values
	l.RLock()
	defer l.RUnlock()

	opts := l.opts
	opts.Fields = copyFields(l.opts.Fields)

	return opts
}

// NewLogger builds a new logger based on options.
func NewLogger(opts ...Option) Logger {
	// Default options
	const defaultCallerSkipCount = 2

	options := Options{
		Level:           InfoLevel,
		Fields:          make(map[string]interface{}),
		Out:             os.Stderr,
		CallerSkipCount: defaultCallerSkipCount,
		Context:         context.Background(),
	}

	l := &defaultLogger{opts: options}
	if err := l.Init(opts...); err != nil {
		l.Log(FatalLevel, err)
	}

	return l
}
