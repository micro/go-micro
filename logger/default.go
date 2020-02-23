package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	dlog "github.com/micro/go-micro/v2/debug/log"
)

type defaultLogger struct {
	opts Options
	err  error
}

// Init(opts...) should only overwrite provided options
func (l *defaultLogger) Init(opts ...Option) error {
	for _, o := range opts {
		o(&l.opts)
	}
	return nil
}

func (l *defaultLogger) String() string {
	return "default"
}

func (l *defaultLogger) Fields(fields map[string]interface{}) Logger {
	l.opts.Fields = fields
	return l
}

func (l *defaultLogger) Error(err error) Logger {
	l.err = err
	return l
}

func (l *defaultLogger) Log(level Level, v ...interface{}) {
	// TODO decide does we need to write message if log level not used?
	if !l.opts.Level.Enabled(level) {
		return
	}

	fields := l.opts.Fields
	fields["level"] = level.String()
	if l.err != nil {
		fields["error"] = l.err.Error()
	}

	rec := dlog.Record{
		Timestamp: time.Now(),
		Message:   fmt.Sprint(v...),
		Metadata:  make(map[string]string, len(fields)),
	}
	for k, v := range fields {
		rec.Metadata[k] = fmt.Sprintf("%v", v)
	}

	dlog.DefaultLog.Write(rec)

	t := rec.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Printf("%s %v\n", t, rec.Message)
}

func (l *defaultLogger) Logf(level Level, format string, v ...interface{}) {
	//	 TODO decide does we need to write message if log level not used?
	if level < l.opts.Level {
		return
	}

	fields := l.opts.Fields
	fields["level"] = level.String()
	if l.err != nil {
		fields["error"] = l.err.Error()
	}

	rec := dlog.Record{
		Timestamp: time.Now(),
		Message:   fmt.Sprintf(format, v...),
		Metadata:  make(map[string]string, len(fields)),
	}
	for k, v := range fields {
		rec.Metadata[k] = fmt.Sprintf("%v", v)
	}

	dlog.DefaultLog.Write(rec)

	t := rec.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Printf("%s %v\n", t, rec.Message)
}

func (n *defaultLogger) Options() Options {
	return n.opts
}

// NewLogger builds a new logger based on options
func NewLogger(opts ...Option) Logger {
	// Default options
	options := Options{
		Level:   InfoLevel,
		Fields:  make(map[string]interface{}),
		Out:     os.Stderr,
		Context: context.Background(),
	}

	l := &defaultLogger{opts: options}
	_ = l.Init(opts...)
	return l
}
