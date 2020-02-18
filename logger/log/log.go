package log

import (
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/logger/basic"
)

// basic Logger is the default global logger.
var globalLogger logger.Logger = basic.NewLogger()

func SetGlobalLogger(logger logger.Logger) {
	globalLogger = logger
}

func SetGlobalLevel(lvl logger.Level) {
	globalLogger.SetLevel(lvl)
}

func Info(args ...interface{}) {
	globalLogger.Log(logger.InfoLevel, "", args, nil)
}
func Infof(template string, args ...interface{}) {
	globalLogger.Log(logger.InfoLevel, template, args, nil)
}
func Infow(msg string, fields logger.Fields) {
	globalLogger.Log(logger.InfoLevel, msg, nil, fields)
}

func Trace(args ...interface{}) {
	globalLogger.Log(logger.TraceLevel, "", args, nil)
}
func Tracef(template string, args ...interface{}) {
	globalLogger.Log(logger.TraceLevel, template, args, nil)
}
func Tracew(msg string, fields logger.Fields) {
	globalLogger.Log(logger.TraceLevel, msg, nil, fields)
}

func Debug(args ...interface{}) {
	globalLogger.Log(logger.DebugLevel, "", args, nil)
}
func Debugf(template string, args ...interface{}) {
	globalLogger.Log(logger.DebugLevel, template, args, nil)
}
func Debugw(msg string, fields logger.Fields) {
	globalLogger.Log(logger.DebugLevel, msg, nil, fields)
}

func Warn(args ...interface{}) {
	globalLogger.Log(logger.WarnLevel, "", args, nil)
}
func Warnf(template string, args ...interface{}) {
	globalLogger.Log(logger.WarnLevel, template, args, nil)
}
func Warnw(msg string, fields logger.Fields) {
	globalLogger.Log(logger.WarnLevel, msg, nil, fields)
}

func Error(args ...interface{}) {
	globalLogger.Log(logger.ErrorLevel, "", args, nil)
}
func Errorf(template string, args ...interface{}) {
	globalLogger.Log(logger.ErrorLevel, template, args, nil)
}
func Errorw(msg string, err error) {
	globalLogger.Error(logger.ErrorLevel, msg, nil, err)
}

func Panic(args ...interface{}) {
	globalLogger.Log(logger.PanicLevel, "", args, nil)
}
func Panicf(template string, args ...interface{}) {
	globalLogger.Log(logger.PanicLevel, template, args, nil)
}
func Panicw(msg string, fields logger.Fields) {
	globalLogger.Log(logger.PanicLevel, msg, nil, fields)
}

func Fatal(args ...interface{}) {
	globalLogger.Log(logger.FatalLevel, "", args, nil)
}
func Fatalf(template string, args ...interface{}) {
	globalLogger.Log(logger.FatalLevel, template, args, nil)
}
func Fatalw(msg string, fields logger.Fields) {
	globalLogger.Log(logger.FatalLevel, msg, nil, fields)
}
