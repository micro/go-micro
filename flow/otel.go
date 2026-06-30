package flow

import (
	"context"
	"time"

	"go-micro.dev/v6/ai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const flowInstrumentationName = "go-micro.dev/v6/flow"

const (
	spanNameFlowRun  = "flow.run"
	spanNameFlowStep = "flow.step"

	AttrFlowRunID     = "flow.run.id"
	AttrFlowParentID  = "flow.run.parent_id"
	AttrFlowName      = "flow.name"
	AttrFlowStepName  = "flow.step.name"
	AttrFlowStatus    = "flow.status"
	AttrFlowAttempts  = "flow.step.attempts"
	AttrFlowLatencyMS = "flow.latency_ms"
	AttrFlowErrorKind = "flow.error.kind"
)

func (f *Flow) tracer() trace.Tracer {
	return f.opts.TraceProvider.Tracer(flowInstrumentationName)
}

func (f *Flow) startRunSpan(ctx context.Context, run Run) (context.Context, func(Run, error)) {
	if f.opts.TraceProvider == nil {
		return ctx, func(Run, error) {}
	}
	ctx, span := f.tracer().Start(ctx, spanNameFlowRun, trace.WithSpanKind(trace.SpanKindInternal), trace.WithAttributes(
		attribute.String(AttrFlowRunID, run.ID),
		attribute.String(AttrFlowParentID, run.ParentID),
		attribute.String(AttrFlowName, f.name),
		attribute.String(AttrFlowStatus, run.Status),
	))
	start := time.Now()
	return ctx, func(done Run, err error) {
		span.SetAttributes(
			attribute.String(AttrFlowStatus, done.Status),
			attribute.Int64(AttrFlowLatencyMS, time.Since(start).Milliseconds()),
		)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.String(AttrFlowErrorKind, string(ai.ClassifyError(err))))
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}
}

func (f *Flow) runStepSpan(ctx context.Context, step Step, in State) (State, int, error) {
	if f.opts.TraceProvider == nil {
		return f.runStep(ctx, step, in)
	}
	info, _ := ai.RunInfoFrom(ctx)
	ctx, span := f.tracer().Start(ctx, spanNameFlowStep, trace.WithAttributes(
		attribute.String(AttrFlowRunID, info.RunID),
		attribute.String(AttrFlowParentID, info.ParentID),
		attribute.String(AttrFlowName, f.name),
		attribute.String(AttrFlowStepName, step.Name),
	))
	start := time.Now()
	out, attempts, err := f.runStep(ctx, step, in)
	span.SetAttributes(
		attribute.Int(AttrFlowAttempts, attempts),
		attribute.Int64(AttrFlowLatencyMS, time.Since(start).Milliseconds()),
	)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String(AttrFlowErrorKind, string(ai.ClassifyError(err))))
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
	return out, attempts, err
}
