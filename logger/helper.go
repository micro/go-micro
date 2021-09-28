package logger

import (
	"context"
	"os"
)

type Helper struct {
	logger Logger
}

func NewHelper(logger Logger) *Helper {
	return &Helper{logger: logger}
}

// Extract always returns valid Helper with logger from context or with DefaultLogger as fallback.
// Can be used in pair with function Inject.
// Example: propagate RequestID to logger in service handler methods.
func Extract(ctx context.Context) *Helper {
	if l, ok := FromContext(ctx); ok {
		return NewHelper(l)
	}

	return NewHelper(DefaultLogger)
}

func (h *Helper) Inject(ctx context.Context) context.Context {
	return NewContext(ctx, h.logger)
}

func (h *Helper) Log(level Level, args ...interface{}) {
	h.logger.Log(level, args...)
}

func (h *Helper) Logf(level Level, template string, args ...interface{}) {
	h.logger.Logf(level, template, args...)
}

func (h *Helper) Info(args ...interface{}) {
	if !h.logger.Options().Level.Enabled(InfoLevel) {
		return
	}
	h.logger.Log(InfoLevel, args...)
}

func (h *Helper) Infof(template string, args ...interface{}) {
	if !h.logger.Options().Level.Enabled(InfoLevel) {
		return
	}
	h.logger.Logf(InfoLevel, template, args...)
}

func (h *Helper) Trace(args ...interface{}) {
	if !h.logger.Options().Level.Enabled(TraceLevel) {
		return
	}
	h.logger.Log(TraceLevel, args...)
}

func (h *Helper) Tracef(template string, args ...interface{}) {
	if !h.logger.Options().Level.Enabled(TraceLevel) {
		return
	}
	h.logger.Logf(TraceLevel, template, args...)
}

func (h *Helper) Debug(args ...interface{}) {
	if !h.logger.Options().Level.Enabled(DebugLevel) {
		return
	}
	h.logger.Log(DebugLevel, args...)
}

func (h *Helper) Debugf(template string, args ...interface{}) {
	if !h.logger.Options().Level.Enabled(DebugLevel) {
		return
	}
	h.logger.Logf(DebugLevel, template, args...)
}

func (h *Helper) Warn(args ...interface{}) {
	if !h.logger.Options().Level.Enabled(WarnLevel) {
		return
	}
	h.logger.Log(WarnLevel, args...)
}

func (h *Helper) Warnf(template string, args ...interface{}) {
	if !h.logger.Options().Level.Enabled(WarnLevel) {
		return
	}
	h.logger.Logf(WarnLevel, template, args...)
}

func (h *Helper) Error(args ...interface{}) {
	if !h.logger.Options().Level.Enabled(ErrorLevel) {
		return
	}
	h.logger.Log(ErrorLevel, args...)
}

func (h *Helper) Errorf(template string, args ...interface{}) {
	if !h.logger.Options().Level.Enabled(ErrorLevel) {
		return
	}
	h.logger.Logf(ErrorLevel, template, args...)
}

func (h *Helper) Fatal(args ...interface{}) {
	if !h.logger.Options().Level.Enabled(FatalLevel) {
		return
	}
	h.logger.Log(FatalLevel, args...)
	os.Exit(1)
}

func (h *Helper) Fatalf(template string, args ...interface{}) {
	if !h.logger.Options().Level.Enabled(FatalLevel) {
		return
	}
	h.logger.Logf(FatalLevel, template, args...)
	os.Exit(1)
}

func (h *Helper) WithError(err error) *Helper {
	return &Helper{logger: h.logger.Fields(map[string]interface{}{"error": err})}
}

func (h *Helper) WithFields(fields map[string]interface{}) *Helper {
	return &Helper{logger: h.logger.Fields(fields)}
}
