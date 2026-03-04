package mcp

import (
	"context"

	"go-micro.dev/v5/metadata"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "go-micro.dev/v5/gateway/mcp"

// Span and attribute names for MCP OpenTelemetry integration.
const (
	spanNameToolCall = "mcp.tool.call"

	// AttrToolName is the tool being called (e.g. "blog.Blog.Create").
	AttrToolName = "mcp.tool.name"
	// AttrTransport is the transport type ("http" or "stdio").
	AttrTransport = "mcp.transport"
	// AttrAccountID is the authenticated account ID.
	AttrAccountID = "mcp.account.id"
	// AttrTraceID is the MCP-specific UUID trace ID (kept for compatibility).
	AttrTraceID = "mcp.trace_id"
	// AttrAuthAllowed records whether auth was granted.
	AttrAuthAllowed = "mcp.auth.allowed"
	// AttrAuthDeniedReason records why auth was denied.
	AttrAuthDeniedReason = "mcp.auth.denied_reason"
	// AttrScopesRequired lists the scopes required by the tool.
	AttrScopesRequired = "mcp.auth.scopes_required"
	// AttrRateLimited records whether the call was rate-limited.
	AttrRateLimited = "mcp.rate_limited"
)

// tracer returns the OTel tracer from the configured provider.
// If no TraceProvider is set, returns a noop tracer.
func (s *Server) tracer() trace.Tracer {
	if s.opts.TraceProvider != nil {
		return s.opts.TraceProvider.Tracer(instrumentationName)
	}
	return trace.NewNoopTracerProvider().Tracer(instrumentationName)
}

// startToolSpan creates a new server span for an MCP tool call.
// It extracts any incoming trace context from metadata and injects
// the new span's context back into metadata for downstream propagation.
func (s *Server) startToolSpan(ctx context.Context, toolName, transport, mcpTraceID string) (context.Context, trace.Span) {
	if s.opts.TraceProvider == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	// Extract incoming trace context from go-micro metadata (if any).
	md, ok := metadata.FromContext(ctx)
	if ok {
		carrier := metadataCarrier(md)
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	}

	ctx, span := s.tracer().Start(ctx, spanNameToolCall,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String(AttrToolName, toolName),
			attribute.String(AttrTransport, transport),
			attribute.String(AttrTraceID, mcpTraceID),
		),
	)

	// Inject OTel trace context back into metadata so downstream
	// RPC calls (via client wrappers) continue the trace.
	if md == nil {
		md = make(metadata.Metadata)
	}
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		md.Set(k, v)
	}
	ctx = metadata.NewContext(ctx, md)

	return ctx, span
}

// setSpanOK marks a span as successful.
func setSpanOK(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// setSpanError records an error on the span.
func setSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// metadataCarrier adapts go-micro metadata to OTel's TextMapCarrier.
type metadataCarrier metadata.Metadata

func (c metadataCarrier) Get(key string) string {
	v, _ := metadata.Metadata(c).Get(key)
	return v
}

func (c metadataCarrier) Set(key, value string) {
	metadata.Metadata(c).Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
