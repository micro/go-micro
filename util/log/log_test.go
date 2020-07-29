package log

import (
	"testing"
)

func TestDebug(t *testing.T) {
	SetLevel(LevelDebug)
	Debug(123)
}
func TestMain(m *testing.M) {
	m.Run()
}
