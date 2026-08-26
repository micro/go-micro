// Package otel bootstraps the global OpenTelemetry tracer provider from
// OTEL_EXPORTER_OTLP_ENDPOINT. Idempotent: safe to call from any
// entrypoint (CLI, agent, flow, service). No env var => noop, no error.
package otel

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

var once sync.Once

var (
	mu sync.Mutex
	tp *sdktrace.TracerProvider
)

// Init wires the OTLP HTTP trace exporter and W3C propagators into the
// global provider when OTEL_EXPORTER_OTLP_ENDPOINT is set. Subsequent
// calls are no-ops.
func Init() {
	once.Do(initOnce)
}

// Shutdown flushes pending spans to the exporter. Safe to call multiple
// times and when Init was a no-op. Call from process exit paths so
// short-lived binaries do not lose spans.
func Shutdown() {
	mu.Lock()
	defer mu.Unlock()
	if tp == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = tp.Shutdown(ctx)
	tp = nil
}

func initOnce() {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		log.Printf("otel: create exporter: %v", err)
		return
	}
	// Name the service so traces do not show up as "unknown_service:<binary>".
	// OTEL_SERVICE_NAME wins; otherwise fall back to the binary basename
	// (the default resource would otherwise label the process unknown_service).
	// ponytail: schemaless so Merge never fights resource.Default()'s
	// bundled schema URL (semconv versions drift across SDK releases).
	attrs := resource.NewSchemaless(semconv.ServiceName(serviceName()))
	res, err := resource.Merge(resource.Default(), attrs)
	if err != nil {
		log.Printf("otel: build resource: %v", err)
		res = attrs
	}
	tp = sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

// serviceName is the resource attribute OTEL_SERVICE_NAME carries into
// traces; without it the SDK labels spans "unknown_service:<executable>".
func serviceName() string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "go-micro"
}
