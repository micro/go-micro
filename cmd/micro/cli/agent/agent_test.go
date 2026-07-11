package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	aiflow "go-micro.dev/v6/flow"
	"go-micro.dev/v6/store"
)

func TestWriteRunIndexJSON(t *testing.T) {
	runs := []goagent.RunSummary{{
		RunID:      "run-1",
		Agent:      "runner",
		StartedAt:  time.Unix(0, 1),
		UpdatedAt:  time.Unix(0, 2),
		DurationMS: 1234,
		Events:     2,
		Status:     "done",
		LastKind:   "tool",
		TraceID:    "1234567890abcdef",
	}}
	var out bytes.Buffer
	if err := writeRunIndex(&out, "runner", runs, true); err != nil {
		t.Fatal(err)
	}
	var got []goagent.RunSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if len(got) != 1 || got[0].RunID != "run-1" || got[0].LastKind != "tool" {
		t.Fatalf("decoded summaries = %#v", got)
	}
}

func TestWriteRunIndexHumanIncludesStatusAndDuration(t *testing.T) {
	runs := []goagent.RunSummary{{
		RunID:      "run-1",
		Agent:      "runner",
		UpdatedAt:  time.Date(2026, 6, 25, 12, 34, 56, 0, time.UTC),
		DurationMS: 1234,
		Events:     2,
		Status:     "done",
		LastKind:   "tool",
		ParentID:   "parent-run",
	}}
	var out bytes.Buffer
	if err := writeRunIndex(&out, "runner", runs, false); err != nil {
		t.Fatal(err)
	}
	line := out.String()
	for _, want := range []string{"run-1", "status=done", "events=2", "duration=1.2s", "last=tool", "parent=parent-run"} {
		if !strings.Contains(line, want) {
			t.Fatalf("human output %q missing %q", line, want)
		}
	}
}

func TestWriteRunIndexIncludesResumeBreadcrumbs(t *testing.T) {
	runs := []goagent.RunSummary{{
		RunID:      "run-failed",
		Agent:      "runner",
		UpdatedAt:  time.Date(2026, 6, 25, 12, 34, 56, 0, time.UTC),
		Events:     3,
		Status:     "error",
		LastKind:   "tool",
		Checkpoint: "failed",
		Stage:      "ask",
	}}
	var out bytes.Buffer
	if err := writeRunIndex(&out, "runner", runs, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"checkpoint=failed", "stage=ask", `micro agent history runner run-failed`, `micro.AgentResume(ctx, agent, "run-failed")`, `micro.ResumeStreamAsk(ctx, agent, "run-failed")`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteRunIndexInputRequiredUsesResumeInput(t *testing.T) {
	runs := []goagent.RunSummary{{RunID: "run-input", Agent: "runner", Status: "running", LastKind: "checkpoint", Checkpoint: "paused", Stage: "input-required"}}
	var out bytes.Buffer
	if err := writeRunIndex(&out, "runner", runs, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{`micro agent history runner run-input`, `micro agent resume-input runner run-input --input <text>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `micro.AgentResume(ctx, agent, "run-input")`) || strings.Contains(got, "ResumeStreamAsk") {
		t.Fatalf("input-required run should point at ResumeInput only, got:\n%s", got)
	}
}

func TestWriteRunHistoryHumanAndJSON(t *testing.T) {
	events := []goagent.RunEvent{{
		Time:      time.Date(2026, 6, 25, 12, 34, 56, 7_000_000, time.UTC),
		RunID:     "run-1",
		Agent:     "runner",
		Kind:      "tool",
		Name:      "probe",
		Provider:  "oteltest",
		Model:     "unit-model",
		LatencyMS: 42,
		Tokens:    ai.Usage{TotalTokens: 5},
		TraceID:   "1234567890abcdef",
		ParentID:  "parent-run",
	}}

	var human bytes.Buffer
	if err := writeRunHistory(&human, "runner", "run-1", events, false); err != nil {
		t.Fatal(err)
	}
	line := human.String()
	for _, want := range []string{"12:34:56.007 tool", "probe", "oteltest/unit-model", "42ms", "tokens=5", "parent=parent-run", "trace=1234567890ab"} {
		if !strings.Contains(line, want) {
			t.Fatalf("human output %q missing %q", line, want)
		}
	}

	var js bytes.Buffer
	if err := writeRunHistory(&js, "runner", "run-1", events, true); err != nil {
		t.Fatal(err)
	}
	var got []goagent.RunEvent
	if err := json.Unmarshal(js.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, js.String())
	}
	if len(got) != 1 || got[0].Name != "probe" || got[0].Tokens.TotalTokens != 5 {
		t.Fatalf("decoded events = %#v", got)
	}
}

func TestResumeInputRunCompletesCheckpointAndInspectSummary(t *testing.T) {
	oldStore := store.DefaultStore
	store.DefaultStore = store.NewMemoryStore()
	t.Cleanup(func() { store.DefaultStore = oldStore })

	ctx := context.Background()
	cp := aiflow.StoreCheckpoint(store.DefaultStore, "runner")
	run := aiflow.Run{ID: "run-input", Flow: "runner", Status: "paused", State: aiflow.State{Stage: "input-required"}, Steps: []aiflow.StepRecord{{Name: "ask", Status: "paused", Error: "Which region?"}}}
	if err := run.State.Set(cliInputPause{OriginalMessage: "deploy", Prompt: "Which region?"}); err != nil {
		t.Fatalf("set pause: %v", err)
	}
	if err := cp.Save(ctx, run); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	var out bytes.Buffer
	if err := resumeInputRun(ctx, &out, "runner", "run-input", "us-east-1"); err != nil {
		t.Fatalf("resumeInputRun: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Recorded input") || !strings.Contains(got, "micro inspect agent runner --limit 1") {
		t.Fatalf("output missing continuation hints:\n%s", got)
	}
	loaded, ok, err := cp.Load(ctx, "run-input")
	if err != nil || !ok {
		t.Fatalf("load checkpoint ok=%v err=%v", ok, err)
	}
	if loaded.Status != "done" || loaded.State.Stage != "done" {
		t.Fatalf("loaded run status/stage = %s/%s, want done/done", loaded.Status, loaded.State.Stage)
	}
	summaries, err := goagent.ListRunSummariesWithOptions(store.DefaultStore, "runner", goagent.RunListOptions{Status: "done"})
	if err != nil {
		t.Fatalf("summaries: %v", err)
	}
	if len(summaries) != 1 || summaries[0].RunID != "run-input" || summaries[0].Status != "done" {
		t.Fatalf("summaries = %#v, want completed run-input", summaries)
	}
}
