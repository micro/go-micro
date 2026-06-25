package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const agentInstrumentationName = "go-micro.dev/v6/agent"

const (
	spanNameRun       = "agent.run"
	spanNameModelCall = "agent.model.call"
	spanNameToolCall  = "agent.tool.call"

	AttrRunID          = "agent.run.id"
	AttrParentRunID    = "agent.run.parent_id"
	AttrAgentName      = "agent.name"
	AttrProvider       = "agent.model.provider"
	AttrModel          = "agent.model.name"
	AttrLatencyMS      = "agent.latency_ms"
	AttrInputTokens    = "agent.tokens.input"
	AttrOutputTokens   = "agent.tokens.output"
	AttrTotalTokens    = "agent.tokens.total"
	AttrToolName       = "agent.tool.name"
	AttrDelegate       = "agent.delegate"
	AttrGuardrailBlock = "agent.guardrail.block"
	AttrRefusal        = "agent.refusal"
)

type RunEvent struct {
	Time      time.Time `json:"time"`
	RunID     string    `json:"run_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Agent     string    `json:"agent"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	Model     string    `json:"model,omitempty"`
	LatencyMS int64     `json:"latency_ms,omitempty"`
	Tokens    Usage     `json:"tokens,omitempty"`
	Refused   string    `json:"refused,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type Usage = ai.Usage

// RunSummary is a compact index entry for a recorded agent run.
type RunSummary struct {
	RunID     string    `json:"run_id"`
	Agent     string    `json:"agent"`
	ParentID  string    `json:"parent_id,omitempty"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Events    int       `json:"events"`
	LastKind  string    `json:"last_kind,omitempty"`
	LastError string    `json:"last_error,omitempty"`
}

func (a *agentImpl) tracer() trace.Tracer {
	return a.opts.TraceProvider.Tracer(agentInstrumentationName)
}

func (a *agentImpl) startRun(ctx context.Context, message string) (context.Context, func(error)) {
	if a.opts.TraceProvider == nil {
		return ctx, func(error) {}
	}
	info, _ := ai.RunInfoFrom(ctx)
	ctx, span := a.tracer().Start(ctx, spanNameRun, trace.WithSpanKind(trace.SpanKindInternal), trace.WithAttributes(
		attribute.String(AttrRunID, info.RunID), attribute.String(AttrParentRunID, info.ParentID), attribute.String(AttrAgentName, info.Agent)))
	start := time.Now()
	a.recordRunEvent(RunEvent{Time: start, RunID: info.RunID, ParentID: info.ParentID, Agent: info.Agent, Kind: "run", Name: message})
	return ctx, func(err error) {
		latency := time.Since(start).Milliseconds()
		span.SetAttributes(attribute.Int64(AttrLatencyMS, latency))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			a.recordRunEvent(RunEvent{Time: time.Now(), RunID: info.RunID, ParentID: info.ParentID, Agent: info.Agent, Kind: "error", LatencyMS: latency, Error: err.Error()})
		} else {
			span.SetStatus(codes.Ok, "")
			a.recordRunEvent(RunEvent{Time: time.Now(), RunID: info.RunID, ParentID: info.ParentID, Agent: info.Agent, Kind: "done", LatencyMS: latency})
		}
		span.End()
	}
}

type tracedModel struct {
	ai.Model
	a *agentImpl
}

func (a *agentImpl) tracedModel(m ai.Model) ai.Model { return &tracedModel{Model: m, a: a} }
func (m *tracedModel) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	if m.a.opts.TraceProvider == nil {
		return m.Model.Generate(ctx, req, opts...)
	}
	info, _ := ai.RunInfoFrom(ctx)
	provider := m.String()
	model := m.Options().Model
	ctx, span := m.a.tracer().Start(ctx, spanNameModelCall, trace.WithAttributes(attribute.String(AttrProvider, provider), attribute.String(AttrModel, model)))
	start := time.Now()
	resp, err := m.Model.Generate(ctx, req, opts...)
	dur := time.Since(start).Milliseconds()
	attrs := []attribute.KeyValue{attribute.Int64(AttrLatencyMS, dur)}
	usage := ai.Usage{}
	if resp != nil {
		usage = resp.Usage
		attrs = appendUsage(attrs, usage)
	}
	span.SetAttributes(attrs...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
	e := RunEvent{Time: time.Now(), RunID: info.RunID, ParentID: info.ParentID, Agent: info.Agent, Kind: "model", Provider: provider, Model: model, LatencyMS: dur, Tokens: usage}
	if err != nil {
		e.Error = err.Error()
	}
	m.a.recordRunEvent(e)
	return resp, err
}

func appendUsage(attrs []attribute.KeyValue, u ai.Usage) []attribute.KeyValue {
	if u.InputTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrInputTokens, u.InputTokens))
	}
	if u.OutputTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrOutputTokens, u.OutputTokens))
	}
	if u.TotalTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrTotalTokens, u.TotalTokens))
	}
	return attrs
}

func (a *agentImpl) traceTool(next ai.ToolHandler) ai.ToolHandler {
	if a.opts.TraceProvider == nil {
		return next
	}
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		info, _ := ai.RunInfoFrom(ctx)
		ctx, span := a.tracer().Start(ctx, spanNameToolCall, trace.WithAttributes(attribute.String(AttrToolName, call.Name), attribute.Bool(AttrDelegate, call.Name == toolDelegate)))
		start := time.Now()
		res := next(ctx, call)
		dur := time.Since(start).Milliseconds()
		attrs := []attribute.KeyValue{attribute.Int64(AttrLatencyMS, dur)}
		if res.Refused != "" {
			attrs = append(attrs, attribute.Bool(AttrGuardrailBlock, true), attribute.String(AttrRefusal, res.Refused))
		}
		span.SetAttributes(attrs...)
		resErr := resultError(res)
		if res.Refused != "" {
			span.SetStatus(codes.Error, res.Refused)
		} else if resErr != "" {
			span.SetStatus(codes.Error, resErr)
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
		a.recordRunEvent(RunEvent{Time: time.Now(), RunID: info.RunID, ParentID: info.ParentID, Agent: info.Agent, Kind: "tool", Name: call.Name, LatencyMS: dur, Refused: res.Refused, Error: resErr})
		return res
	}
}

func resultError(res ai.ToolResult) string {
	if m, ok := res.Value.(map[string]string); ok {
		return m["error"]
	}
	if m, ok := res.Value.(map[string]any); ok {
		if err, _ := m["error"].(string); err != "" {
			return err
		}
	}
	return ""
}

func (a *agentImpl) recordRunEvent(e RunEvent) {
	if a.opts.TraceProvider == nil || e.RunID == "" {
		return
	}
	b, _ := json.Marshal(e)
	key := fmt.Sprintf("runs/%s/%020d-%s", e.RunID, e.Time.UnixNano(), e.Kind)
	_ = a.stateStore().Write(&store.Record{Key: key, Value: b})
}

// ListRunSummaries returns a deterministic summary of recorded runs for agentName.
func ListRunSummaries(s store.Store, agentName string) ([]RunSummary, error) {
	st := store.Scope(s, "agent", agentName)
	keys, err := st.List(store.ListPrefix("runs/"))
	if err != nil {
		return nil, err
	}
	runs := map[string]bool{}
	for _, k := range keys {
		parts := strings.Split(k, "/")
		if len(parts) >= 2 && parts[1] != "" {
			runs[parts[1]] = true
		}
	}
	ids := make([]string, 0, len(runs))
	for id := range runs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	summaries := make([]RunSummary, 0, len(ids))
	for _, id := range ids {
		events, err := LoadRunEvents(s, agentName, id)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			continue
		}
		first := events[0]
		last := events[len(events)-1]
		summary := RunSummary{
			RunID:     id,
			Agent:     first.Agent,
			ParentID:  first.ParentID,
			StartedAt: first.Time,
			UpdatedAt: last.Time,
			Events:    len(events),
			LastKind:  last.Kind,
			LastError: last.Error,
		}
		for _, e := range events {
			if e.Agent != "" {
				summary.Agent = e.Agent
			}
			if e.ParentID != "" {
				summary.ParentID = e.ParentID
			}
			if e.Error != "" {
				summary.LastError = e.Error
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func LoadRunEvents(s store.Store, agentName, runID string) ([]RunEvent, error) {
	st := store.Scope(s, "agent", agentName)
	keys, err := st.List(store.ListPrefix("runs/" + runID + "/"))
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	events := make([]RunEvent, 0, len(keys))
	for _, k := range keys {
		recs, err := st.Read(k)
		if err != nil || len(recs) == 0 {
			continue
		}
		var e RunEvent
		if json.Unmarshal(recs[0].Value, &e) == nil {
			events = append(events, e)
		}
	}
	return events, nil
}
