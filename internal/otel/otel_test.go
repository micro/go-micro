package otel

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestShutdownFlushes(t *testing.T) {
	exp, err := stdouttrace.New(stdouttrace.WithWriter(testWriter{t}))
	if err != nil {
		t.Fatal(err)
	}
	// simulate initOnce by wiring a batcher provider directly
	tp = sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))
	otel.SetTracerProvider(tp)

	_, span := otel.Tracer("t").Start(context.Background(), "shutdown-probe")
	span.End()
	Shutdown()
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("EXPORTED: %s", p)
	return len(p), nil
}

func TestServiceName(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "support")
	if got := serviceName(); got != "support" {
		t.Fatalf("serviceName() = %q, want %q", got, "support")
	}
	os.Unsetenv("OTEL_SERVICE_NAME")
	if got := serviceName(); got != filepath.Base(os.Args[0]) {
		t.Fatalf("serviceName() = %q, want binary basename %q", got, filepath.Base(os.Args[0]))
	}
}
