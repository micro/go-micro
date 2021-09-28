package zap

import (
	"testing"

	"github.com/asim/go-micro/v3/logger"
)

func TestName(t *testing.T) {
	l, err := NewLogger()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != "zap" {
		t.Errorf("name is error %s", l.String())
	}

	t.Logf("test logger name: %s", l.String())
}

func TestLogf(t *testing.T) {
	// skip is 2, because we call logger through logger package
	l, err := NewLogger(logger.WithCallerSkipCount(2))
	if err != nil {
		t.Fatal(err)
	}

	logger.DefaultLogger = l
	logger.Logf(logger.InfoLevel, "test logf: %s", "name")
}

func TestSetLevel(t *testing.T) {
	// skip is 1, because we call logger directly
	l, err := NewLogger(logger.WithCallerSkipCount(1))
	if err != nil {
		t.Fatal(err)
	}
	logger.DefaultLogger = l

	logger.Init(logger.WithLevel(logger.DebugLevel))
	l.Logf(logger.DebugLevel, "test show debug: %s", "debug msg")

	logger.Init(logger.WithLevel(logger.InfoLevel))
	l.Logf(logger.DebugLevel, "test non-show debug: %s", "debug msg")
}
