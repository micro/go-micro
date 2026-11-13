package inspect

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	goagent "go-micro.dev/v6/agent"
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
	for _, want := range []string{`Agent "support" run "run-1"`, "schema=1", "status=timeout", "events=2", "duration_ms=2000", "trace=trace-1", "Origin", "flow=daily-ops", "step=summarize", "dispatch=schedule", `trigger="daily-review"`, "Timeline", "kind=run", "input_chars=18", "kind=error", "error_kind=timeout", `error="deadline exceeded"`} {
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
