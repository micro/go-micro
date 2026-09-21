package wrapper

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go-micro.dev/v6/logger"
	"go-micro.dev/v6/server"
)

type errorLog struct {
	logger.Logger
	fields  map[string]interface{}
	message string
}

func (l *errorLog) Fields(f map[string]interface{}) logger.Logger { l.fields = f; return l }
func (l *errorLog) Logf(_ logger.Level, format string, args ...interface{}) {
	l.message = fmt.Sprintf(format, args...)
}

type errorRequest struct{ server.Request }

func (errorRequest) Service() string  { return "service" }
func (errorRequest) Endpoint() string { return "Handler.Call" }

func TestLogRawErrorsPreservesFailure(t *testing.T) {
	cause := errors.New("database unavailable")
	log := new(errorLog)
	handler := LogRawErrors(log)(func(context.Context, server.Request, interface{}) error { return cause })
	if got := handler(context.Background(), errorRequest{}, nil); got != cause {
		t.Fatal("wrapper changed error")
	}
	if log.message != "RPC handler failed: database unavailable" || log.fields["structured_error"] != false || log.fields["endpoint"] != "Handler.Call" {
		t.Fatalf("log=%+v %q", log.fields, log.message)
	}
}
