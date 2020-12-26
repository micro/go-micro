package apex

import (
	apexLog "github.com/apex/log"
	log "github.com/micro/go-micro/v2/logger"
)

var (
	lvl = log.InfoLevel
)

type logger struct {
	apexLog.Interface
	opts Options
}

// Fields set fields to always be logged
func (l *logger) Fields(fields map[string]interface{}) log.Logger {
	data := make(apexLog.Fields, len(fields))
	for k, v := range fields {
		data[k] = v
	}
	return newLogger(l.WithFields(data))
}

// Init initializes options
func (l *logger) Init(opts ...log.Option) error {
	options := &Options{}
	for _, o := range opts {
		o(&options.Options)
	}

	if options.Context != nil {

		if h, ok := options.Context.Value(handlerKey{}).(apexLog.Handler); ok {
			apexLog.SetHandler(h)
		}

		if lvl, ok := options.Context.Value(levelKey{}).(log.Level); ok {
			l.setLogLevel(lvl)
		}
	}

	return nil
}

func (l *logger) Options() log.Options {
	// FIXME: How to return full opts?
	return l.opts.Options
}

func (l *logger) setLogLevel(level log.Level) {
	lvl = level
	apexLog.SetLevel(convertToApexLevel(level))
}

// Log inserts a log entry.  Arguments may be handled in the manner
// of fmt.Print, but the underlying logger may also decide to handle
// them differently.
func (l *logger) Log(level log.Level, v ...interface{}) {
	l.Logf(level, "%s", v)
}

// Logf insets a log entry.  Arguments are handled in the manner of
// fmt.Printf.
func (l *logger) Logf(level log.Level, format string, v ...interface{}) {
	apexlevel := convertToApexLevel(level)
	switch apexlevel {
	case apexLog.FatalLevel:
		l.Interface.Fatalf(format, v...)
	case apexLog.ErrorLevel:
		l.Interface.Errorf(format, v...)
	case apexLog.WarnLevel:
		l.Interface.Warnf(format, v...)
	case apexLog.DebugLevel:
		l.Interface.Debugf(format, v...)
	default:
		l.Interface.Infof(format, v...)
	}
}

// String returns the name of logger
func (l *logger) String() string {
	return "apex"
}

func newLogger(logInstance apexLog.Interface) log.Logger {
	return &logger{
		logInstance,
		Options{
			log.Options{
				Level: log.InfoLevel,
			},
		},
	}
}

// New returns a new ApexLogger instance
func New(opts ...log.Option) log.Logger {
	l := newLogger(apexLog.Log)
	_ = l.Init(opts...)
	return l
}

func convertToApexLevel(level log.Level) apexLog.Level {
	switch level {
	case log.DebugLevel:
		return apexLog.DebugLevel
	case log.InfoLevel:
		return apexLog.InfoLevel
	case log.WarnLevel:
		return apexLog.WarnLevel
	case log.ErrorLevel:
		return apexLog.ErrorLevel
	case log.FatalLevel:
		return apexLog.FatalLevel
	case log.TraceLevel:
		return apexLog.DebugLevel
	default:
		return apexLog.InfoLevel
	}
}

func convertLevel(level apexLog.Level) log.Level {
	switch level {
	case apexLog.DebugLevel:
		return log.DebugLevel
	case apexLog.InfoLevel:
		return log.InfoLevel
	case apexLog.WarnLevel:
		return log.WarnLevel
	case apexLog.ErrorLevel:
		return log.ErrorLevel
	case apexLog.FatalLevel:
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}
