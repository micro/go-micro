package logger

type Helper struct {
	Logger
}

func NewHelper(log Logger) *Helper {
	return &Helper{log}
}

func (h *Helper) Info(args ...interface{}) {
	h.Logger.Log(InfoLevel, args...)
}

func (h *Helper) Infof(template string, args ...interface{}) {
	h.Logger.Logf(InfoLevel, template, args...)
}

func (h *Helper) Trace(args ...interface{}) {
	h.Logger.Log(TraceLevel, args...)
}

func (h *Helper) Tracef(template string, args ...interface{}) {
	h.Logger.Logf(TraceLevel, template, args...)
}

func (h *Helper) Debug(args ...interface{}) {
	h.Logger.Log(DebugLevel, args...)
}

func (h *Helper) Debugf(template string, args ...interface{}) {
	h.Logger.Logf(DebugLevel, template, args...)
}

func (h *Helper) Warn(args ...interface{}) {
	h.Logger.Log(WarnLevel, args...)
}

func (h *Helper) Warnf(template string, args ...interface{}) {
	h.Logger.Logf(WarnLevel, template, args...)
}

func (h *Helper) Error(args ...interface{}) {
	h.Logger.Log(ErrorLevel, args...)
}

func (h *Helper) Errorf(template string, args ...interface{}) {
	h.Logger.Logf(ErrorLevel, template, args...)
}

func (h *Helper) Fatal(args ...interface{}) {
	h.Logger.Log(ErrorLevel, args...)
}

func (h *Helper) Fatalf(template string, args ...interface{}) {
	h.Logger.Logf(ErrorLevel, template, args...)
}

func (h *Helper) WithError(err error) *Helper {
	return &Helper{h.Logger.Fields(map[string]interface{}{"error": err})}
}
