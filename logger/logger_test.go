package logger

import (
	"context"
	"testing"
)

func TestLogger(t *testing.T) {
	l := NewLogger(WithLevel(TraceLevel), WithCallerSkipCount(2))

	h1 := NewHelper(l).WithFields(map[string]interface{}{"key1": "val1"})
	h1.Log(TraceLevel, "simple log before trace_msg1")
	h1.Trace("trace_msg1")
	h1.Log(TraceLevel, "simple log after trace_msg1")
	h1.Warn("warn_msg1")

	h2 := NewHelper(l).WithFields(map[string]interface{}{"key2": "val2"})
	h2.Logf(TraceLevel, "formatted log before trace_msg%s", "2")
	h2.Trace("trace_msg2")
	h2.Logf(TraceLevel, "formatted log after trace_msg%s", "2")
	h2.Warn("warn_msg2")

	l = NewLogger(WithLevel(TraceLevel), WithCallerSkipCount(1))
	l.Fields(map[string]interface{}{"key3": "val4"}).Log(InfoLevel, "test_msg")
}

func TestExtract(t *testing.T) {
	l := NewLogger(WithLevel(TraceLevel), WithCallerSkipCount(2)).Fields(map[string]interface{}{"requestID": "req-1"})

	ctx := NewContext(context.Background(), l)

	Info("info message without request ID")
	Extract(ctx).Info("info message with request ID")
}
