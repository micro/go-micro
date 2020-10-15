package logger

import (
	"testing"
)

func TestLogger(t *testing.T) {
	l := NewLogger(WithLevel(TraceLevel))
	h1 := NewHelper(l).WithFields(map[string]interface{}{"key1": "val1"})
	h1.Trace("trace_msg1")
	h1.Warn("warn_msg1")

	h2 := NewHelper(l).WithFields(map[string]interface{}{"key2": "val2"})
	h2.Trace("trace_msg2")
	h2.Warn("warn_msg2")

	l.Fields(map[string]interface{}{"key3": "val4"}).Log(InfoLevel, "test_msg")
}
