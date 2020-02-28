package logger

import "testing"

func TestLogger(t *testing.T) {
	l := NewLogger(WithLevel(TraceLevel))
	h := NewHelper(l)
	h.Trace("trace level")
}
