// Package otel bootstraps the global OpenTelemetry tracer provider from
// OTEL_EXPORTER_OTLP_ENDPOINT. It is a lightweight facade: the actual OTLP
// wiring lives behind the go-micro.dev/v6/otel backend, which registers
// itself here via Register and is linked by blank import. Core packages
// (service, agent, flow, and the micro constructor) call Init and Shutdown
// unconditionally; with no backend linked both are no-ops and nothing heavy
// (the otel SDK, otlptrace exporter, or grpc) enters the build.
//
// Runtime behavior is unchanged: no env var => noop, no error; idempotent.
package otel

var (
	build func()
	flush func()
)

// Register mounts the OTLP backend. Called from go-micro.dev/v6/otel's init;
// exported only so that package can install itself.
func Register(initFn, shutdownFn func()) {
	build = initFn
	flush = shutdownFn
}

// Init wires the global tracer provider from OTEL_EXPORTER_OTLP_ENDPOINT
// when an OTLP backend is linked. Subsequent calls are no-ops. No backend
// and no env var are both no-ops.
func Init() {
	if build != nil {
		build()
	}
}

// Shutdown flushes pending spans to the exporter. Safe to call multiple
// times and when Init was a no-op.
func Shutdown() {
	if flush != nil {
		flush()
	}
}
