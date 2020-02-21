package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	if !l.opts.Level.Enabled(level) {
		return
	}
	msg := fmt.Sprint(v...)

	fields := l.opts.Fields
	fields["level"] = level.String()
	fields["message"] = msg
	if l.err != nil {
		fields["error"] = l.err.Error()
	}

	enc := json.NewEncoder(l.opts.Out)

	if err := enc.Encode(fields); err != nil {
		log.Fatal(err)
	}
}

func (l *defaultLogger) Logf(level Level, format string, v ...interface{}) {
	if level < l.opts.Level {
		return
	}
	msg := fmt.Sprintf(format, v...)

	fields := l.opts.Fields
	fields["level"] = level.String()
	fields["message"] = msg
	if l.err != nil {
		fields["error"] = l.err.Error()
	}

	enc := json.NewEncoder(l.opts.Out)

	if err := enc.Encode(fields); err != nil {
		log.Fatal(err)
	}

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
