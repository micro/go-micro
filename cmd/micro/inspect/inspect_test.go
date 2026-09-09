package inspect

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	goagent "go-micro.dev/v6/agent"
	aiflow "go-micro.dev/v6/flow"
)

func TestWriteAgentInspectionIncludesActionableBreadcrumbs(t *testing.T) {
	runs := []goagent.RunSummary{{RunID: "run-1", Status: "auth", Events: 4, LastKind: "model", LastError: "invalid API key", LastErrorKind: "auth", TraceID: "1234567890abcdef", Checkpoint: "failed", Stage: "ask", Spent: 7}}
	var out bytes.Buffer
	if err := writeAgentInspection(&out, "support", runs, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"Agent \"support\" runs", "run-1", "status=auth", "events=4", "last=model", "checkpoint=failed", "stage=ask", "error_kind=auth", `error="invalid API key"`, "trace=1234567890ab", "spent=7"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteAgentInspectionIncludesInputResumeBreadcrumb(t *testing.T) {
	runs := []goagent.RunSummary{{RunID: "run-input", Status: "running", Events: 3, LastKind: "checkpoint", Checkpoint: "paused", Stage: "input-required"}}
	var out bytes.Buffer
	if err := writeAgentInspection(&out, "support", runs, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"checkpoint=paused", "stage=input-required", `micro agent history support run-input`, `micro agent resume-input support run-input --input <text>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `micro.AgentResume(ctx, agent, "run-input")`) || strings.Contains(got, "ResumeStreamAsk") {
		t.Fatalf("input-required run should point at ResumeInput only, got:\n%s", got)
	}
}

func TestWriteAgentInspectionEmptyStateNamesInspectCommand(t *testing.T) {
	var out bytes.Buffer
	if err := writeAgentInspection(&out, "support", nil, false); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "micro inspect agent support") {
		t.Fatalf("empty state missing next step: %q", got)
	}
}

func TestWriteAgentRunRecordIncludesSummaryAndTimeline(t *testing.T) {
	start := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	record := goagent.RunRecord{
		SchemaVersion: goagent.RunRecordSchemaVersion,
		Summary: goagent.RunSummary{
			RunID: "run-1", Agent: "support", Status: "timeout", Events: 2,
			StartedAt: start, UpdatedAt: start.Add(2 * time.Second), DurationMS: 2000,
			TraceID: "trace-1", ParentID: "flow-run-1", Flow: "daily-ops", Step: "summarize",
			Dispatch: "schedule", Trigger: "daily-review", LastError: "deadline exceeded", LastErrorKind: "timeout",
		},
		Events: []goagent.RunEvent{
			{Time: start, RunID: "run-1", Agent: "support", Kind: "run", InputChars: 18},
			{Time: start.Add(2 * time.Second), RunID: "run-1", Agent: "support", Kind: "error", Error: "deadline exceeded", ErrorKind: "timeout"},
		},
	}
	var out bytes.Buffer
	if err := writeAgentRunRecord(&out, record, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{`Agent "support" run "run-1"`, "schema=1", "status=timeout", "events=2", "duration_ms=2000", "trace=trace-1", "Origin", "flow=daily-ops", "step=summarize", "dispatch=schedule", `trigger="daily-review"`, "micro inspect flow daily-ops --run flow-run-1", "Timeline", "kind=run", "input_chars=18", "kind=error", "error_kind=timeout", `error="deadline exceeded"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteAgentRunRecordJSONPreservesSchema(t *testing.T) {
	record := goagent.RunRecord{
		SchemaVersion: goagent.RunRecordSchemaVersion,
		Summary:       goagent.RunSummary{RunID: "run-1", Agent: "support"},
		Events:        []goagent.RunEvent{},
	}
	var out bytes.Buffer
	if err := writeAgentRunRecord(&out, record, true); err != nil {
		t.Fatal(err)
	}
	var got goagent.RunRecord
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.SchemaVersion != goagent.RunRecordSchemaVersion || got.Summary.RunID != "run-1" || got.Events == nil {
		t.Fatalf("decoded record = %#v", got)
	}
}

func TestWriteFlowInspectionIncludesFailedStepBreadcrumb(t *testing.T) {
	runs := []aiflow.Run{{ID: "1234567890abcdef", Status: "failed", State: aiflow.State{Stage: "charge"}, Steps: []aiflow.StepRecord{{Name: "charge", Status: "failed", Error: "card declined"}}}}
	var out bytes.Buffer
	if err := writeFlowInspection(&out, "checkout", runs, false, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"Flow \"checkout\" runs", "1234567890ab", "status=failed", "stage=charge", "steps=1", `error="card declined"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteFlowInspectionJSON(t *testing.T) {
	runs := []aiflow.Run{{ID: "run-1", Flow: "checkout", Status: "done"}}
	var out bytes.Buffer
	if err := writeFlowInspection(&out, "checkout", runs, true, false); err != nil {
		t.Fatal(err)
	}
	var got []aiflow.Run
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if len(got) != 1 || got[0].ID != "run-1" || got[0].Status != "done" {
		t.Fatalf("decoded runs = %+v", got)
	}
}

func TestWriteFlowRunRecordIncludesTargetsAndChildBreadcrumb(t *testing.T) {
	start := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	record := aiflow.RunRecord{
		SchemaVersion: aiflow.RunRecordSchemaVersion,
		Run: aiflow.Run{
			ID: "flow-run-1", Flow: "checkout", Status: "done", Dispatch: "schedule", Trigger: "nightly",
			Started: start, Updated: start.Add(time.Second),
			Steps: []aiflow.StepRecord{
				{Name: "charge", Status: "done", Attempts: 1, Service: "payments", Endpoint: "Payments.Charge"},
				{Name: "notify", Status: "done", Attempts: 1, Agent: "comms", ChildRunID: "agent-run-1"},
			},
		},
	}
	var out bytes.Buffer
	if err := writeFlowRunRecord(&out, record, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{`Flow "checkout" run "flow-run-1"`, "schema=1", "status=done", "dispatch=schedule", `trigger="nightly"`, "Steps", "service=payments", "endpoint=Payments.Charge", "agent=comms", "child_run=agent-run-1", "micro inspect agent comms --run agent-run-1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteFlowRunRecordJSONPreservesSchema(t *testing.T) {
	record := aiflow.RunRecord{SchemaVersion: aiflow.RunRecordSchemaVersion, Run: aiflow.Run{ID: "flow-run-1", Flow: "checkout"}}
	var out bytes.Buffer
	if err := writeFlowRunRecord(&out, record, true); err != nil {
		t.Fatal(err)
	}
	var got aiflow.RunRecord
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.SchemaVersion != aiflow.RunRecordSchemaVersion || got.Run.ID != "flow-run-1" {
		t.Fatalf("decoded record = %#v", got)
	}
}
