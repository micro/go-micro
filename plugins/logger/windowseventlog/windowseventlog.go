// +build windows

package windowseventlog

import (
	"context"
	"fmt"
	"os"

	"github.com/asim/go-micro/v3/logger"
	"golang.org/x/sys/windows/svc/eventlog"
)

type eventLogger struct {
	elog *eventlog.Log
	opts Options
}

func NewLogger(opts ...logger.Option) *eventLogger {

	options := Options{
		Options: logger.Options{
			Level:   logger.InfoLevel,
			Fields:  make(map[string]interface{}),
			Out:     os.Stderr,
			Context: context.Background(),
		},
		Src: "go-micro logger",
		Eid: 1,
	}

	l := &eventLogger{
		opts: options,
	}

	_ = l.Init(opts...)

	elog, err := eventlog.Open(l.opts.Src)
	if err == nil {
		l.elog = elog
	}

	return l
}

func (l *eventLogger) Init(opts ...logger.Option) error {

	for _, o := range opts {
		o(&l.opts.Options)
	}

	if srcname, ok := l.opts.Context.Value(src{}).(string); ok {
		l.opts.Src = srcname
	}

	if neweid, ok := l.opts.Context.Value(eid{}).(uint32); ok {
		l.opts.Eid = neweid
	}

	err := eventlog.InstallAsEventCreate(l.opts.Src, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		return err
	}

	if l.elog == nil {
		elog, err := eventlog.Open(l.opts.Src)
		if err != nil {
			return err
		}
		l.elog = elog
	}

	return nil
}

func (l *eventLogger) Options() logger.Options {
	return l.opts.Options
}

func (l *eventLogger) Fields(fields map[string]interface{}) logger.Logger {
	return l
}

func (l *eventLogger) Log(level logger.Level, v ...interface{}) {

	l.Logf(level, "%v", v...)

}

func (l *eventLogger) Logf(level logger.Level, format string, v ...interface{}) {

	msg := fmt.Sprintf(format, v...)

	switch level {
	case logger.TraceLevel:
		_ = l.elog.Info(l.opts.Eid, msg)
	case logger.DebugLevel:
		_ = l.elog.Info(l.opts.Eid, msg)
	case logger.InfoLevel:
		_ = l.elog.Info(l.opts.Eid, msg)
	case logger.WarnLevel:
		_ = l.elog.Warning(l.opts.Eid, msg)
	case logger.ErrorLevel:
		_ = l.elog.Error(l.opts.Eid, msg)
	case logger.FatalLevel:
		_ = l.elog.Error(l.opts.Eid, msg)
	}

}

func (l *eventLogger) String() string {
	return "windowseventlog"
}
