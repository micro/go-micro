package zerolog

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/asim/go-micro/v3/logger"
)

type Mode uint8

const (
	Production Mode = iota
	Development
)

type zeroLogger struct {
	zLog zerolog.Logger
	opts Options
}

func (l *zeroLogger) Init(opts ...logger.Option) error {
	for _, o := range opts {
		o(&l.opts.Options)
	}

	if hs, ok := l.opts.Context.Value(hooksKey{}).([]zerolog.Hook); ok {
		l.opts.Hooks = hs
	}
	if tf, ok := l.opts.Context.Value(timeFormatKey{}).(string); ok {
		l.opts.TimeFormat = tf
	}
	if exitFunction, ok := l.opts.Context.Value(exitKey{}).(func(int)); ok {
		l.opts.ExitFunc = exitFunction
	}
	if caller, ok := l.opts.Context.Value(reportCallerKey{}).(bool); ok && caller {
		l.opts.ReportCaller = caller
	}
	if useDefault, ok := l.opts.Context.Value(useAsDefaultKey{}).(bool); ok && useDefault {
		l.opts.UseAsDefault = useDefault
	}
	if devMode, ok := l.opts.Context.Value(developmentModeKey{}).(bool); ok && devMode {
		l.opts.Mode = Development
	}
	if prodMode, ok := l.opts.Context.Value(productionModeKey{}).(bool); ok && prodMode {
		l.opts.Mode = Production
	}

	// RESET
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.ErrorStackMarshaler = nil
	zerolog.CallerSkipFrameCount = 4

	switch l.opts.Mode {
	case Development:
		zerolog.ErrorStackMarshaler = func(err error) interface{} {
			fmt.Println(string(debug.Stack()))
			return nil
		}
		consOut := zerolog.NewConsoleWriter(
			func(w *zerolog.ConsoleWriter) {
				if len(l.opts.TimeFormat) > 0 {
					w.TimeFormat = l.opts.TimeFormat
				}
				w.Out = l.opts.Out
				w.NoColor = false
			},
		)
		//level = logger.DebugLevel
		l.zLog = zerolog.New(consOut).
			Level(zerolog.DebugLevel).
			With().Timestamp().Stack().Logger()
	default: // Production
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		l.zLog = zerolog.New(l.opts.Out).
			Level(zerolog.InfoLevel).
			With().Timestamp().Stack().Logger()
	}

	// Set log Level if not default
	if l.opts.Level != 100 {
		zerolog.SetGlobalLevel(loggerToZerologLevel(l.opts.Level))
		l.zLog = l.zLog.Level(loggerToZerologLevel(l.opts.Level))
	}

	// Reporting caller
	if l.opts.ReportCaller {
		l.zLog = l.zLog.With().Caller().Logger()
	}
	
	// Adding hooks if exist
	for _, hook := range l.opts.Hooks {
		l.zLog = l.zLog.Hook(hook)
	}

	// Setting timeFormat
	if len(l.opts.TimeFormat) > 0 {
		zerolog.TimeFieldFormat = l.opts.TimeFormat
	}

	// Adding seed fields if exist
	if l.opts.Fields != nil {
		l.zLog = l.zLog.With().Fields(l.opts.Fields).Logger()
	}

	// Also set it as zerolog's Default logger
	if l.opts.UseAsDefault {
		zlog.Logger = l.zLog
	}

	return nil
}

func (l *zeroLogger) Fields(fields map[string]interface{}) logger.Logger {
	l.zLog = l.zLog.With().Fields(fields).Logger()
	return l
}

func (l *zeroLogger) Error(err error) logger.Logger {
	l.zLog = l.zLog.With().Fields(map[string]interface{}{zerolog.ErrorFieldName: err}).Logger()
	return l
}

func (l *zeroLogger) Log(level logger.Level, args ...interface{}) {
	msg := fmt.Sprint(args...)
	l.zLog.WithLevel(loggerToZerologLevel(level)).Msg(msg)
	// Invoke os.Exit because unlike zerolog.Logger.Fatal zerolog.Logger.WithLevel won't stop the execution.
	if level == logger.FatalLevel {
		l.opts.ExitFunc(1)
	}
}

func (l *zeroLogger) Logf(level logger.Level, format string, args ...interface{}) {
	l.zLog.WithLevel(loggerToZerologLevel(level)).Msgf(format, args...)
	// Invoke os.Exit because unlike zerolog.Logger.Fatal zerolog.Logger.WithLevel won't stop the execution.
	if level == logger.FatalLevel {
		l.opts.ExitFunc(1)
	}
}

func (l *zeroLogger) String() string {
	return "zerolog"
}

func (l *zeroLogger) Options() logger.Options {
	// FIXME: How to return full opts?
	return l.opts.Options
}

// NewLogger builds a new logger based on options
func NewLogger(opts ...logger.Option) logger.Logger {
	// Default options
	options := Options{
		Options: logger.Options{
			Level:   100,
			Fields:  make(map[string]interface{}),
			Out:     os.Stderr,
			Context: context.Background(),
		},
		ReportCaller: false,
		UseAsDefault: false,
		Mode:         Production,
		ExitFunc:     os.Exit,
	}

	l := &zeroLogger{opts: options}
	_ = l.Init(opts...)
	return l
}

func loggerToZerologLevel(level logger.Level) zerolog.Level {
	switch level {
	case logger.TraceLevel:
		return zerolog.TraceLevel
	case logger.DebugLevel:
		return zerolog.DebugLevel
	case logger.InfoLevel:
		return zerolog.InfoLevel
	case logger.WarnLevel:
		return zerolog.WarnLevel
	case logger.ErrorLevel:
		return zerolog.ErrorLevel
	case logger.FatalLevel:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func ZerologToLoggerLevel(level zerolog.Level) logger.Level {
	switch level {
	case zerolog.TraceLevel:
		return logger.TraceLevel
	case zerolog.DebugLevel:
		return logger.DebugLevel
	case zerolog.InfoLevel:
		return logger.InfoLevel
	case zerolog.WarnLevel:
		return logger.WarnLevel
	case zerolog.ErrorLevel:
		return logger.ErrorLevel
	case zerolog.FatalLevel:
		return logger.FatalLevel
	default:
		return logger.InfoLevel
	}
}
