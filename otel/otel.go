// Package otel links the OTLP HTTP trace exporter into the builtin
// OpenTelemetry facade.
//
// Blank-import it to enable automatic trace export for services, agents,
// and flows. The go-micro binaries do this from cmd/micro; library binaries
// skip it to keep the otel SDK, otlptrace exporter, and grpc out of the
// build:
//
//	import _ "go-micro.dev/v6/otel"
//
// Behavior matches the previous implicit setup: when
// OTEL_EXPORTER_OTLP_ENDPOINT is set, the global tracer provider is built
// and flushed on shutdown; otherwise everything is a no-op.
package otel

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	builtin "go-micro.dev/v6/internal/otel"

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

func init() {
	builtin.Register(buildOnce, shutdown)
}

// buildOnce wires the OTLP HTTP trace exporter and W3C propagators into the
// global provider when OTEL_EXPORTER_OTLP_ENDPOINT is set. Subsequent
// calls are no-ops (guarded by once).
func buildOnce() {
	once.Do(initOnce)
}

// shutdown flushes pending spans to the exporter. Safe to call multiple
// times and when initOnce was a no-op. Call from process exit paths so
// short-lived binaries do not lose spans.
func shutdown() {
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
