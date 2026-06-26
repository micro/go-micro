package flow

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	aiflow "go-micro.dev/v6/flow"
)

func TestWriteFlowRunsIncludesStepDetails(t *testing.T) {
	updated := time.Date(2026, 6, 24, 12, 30, 0, 0, time.UTC)
	runs := []aiflow.Run{{
		ID:      "1234567890abcdef",
		Status:  "failed",
		Updated: updated,
		State:   aiflow.State{Stage: "charge"},
		Steps: []aiflow.StepRecord{
			{Name: "reserve", Status: "done", Attempts: 1},
			{Name: "charge", Status: "failed", Attempts: 3, Error: "card declined"},
		},
	}}

	var out bytes.Buffer
	if err := writeFlowRuns(&out, runs, false); err != nil {
		t.Fatalf("writeFlowRuns: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"12345678  failed   stage=charge",
		"updated=2026-06-24T12:30:00Z",
		"- reserve      done        attempts=1",
		`- charge       failed      attempts=3 error="card declined"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteFlowRunsJSON(t *testing.T) {
	runs := []aiflow.Run{{ID: "run-1", Flow: "checkout", Status: "done"}}

	var out bytes.Buffer
	if err := writeFlowRuns(&out, runs, true); err != nil {
		t.Fatalf("writeFlowRuns: %v", err)
	}
	var got []aiflow.Run
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if len(got) != 1 || got[0].ID != "run-1" || got[0].Flow != "checkout" || got[0].Status != "done" {
		t.Fatalf("decoded runs = %+v", got)
	}
}

func TestPendingFlowRunsFiltersCompletedRuns(t *testing.T) {
	runs := []aiflow.Run{
		{ID: "run-1", Status: "done"},
		{ID: "run-2", Status: "failed"},
		{ID: "run-3", Status: "running"},
	}

	got := pendingFlowRuns(runs)
	if len(got) != 2 {
		t.Fatalf("pendingFlowRuns returned %d runs, want 2: %+v", len(got), got)
	}
	if got[0].ID != "run-2" || got[1].ID != "run-3" {
		t.Fatalf("pending runs = %+v", got)
	}
}

func TestFilterFlowRunsStatus(t *testing.T) {
	runs := []aiflow.Run{
		{ID: "run-1", Status: "done"},
		{ID: "run-2", Status: "failed"},
		{ID: "run-3", Status: "running"},
		{ID: "run-4", Status: "failed"},
	}

	got := filterFlowRuns(runs, flowRunOptions{Status: "failed"})
	if len(got) != 2 {
		t.Fatalf("filterFlowRuns returned %d runs, want 2: %+v", len(got), got)
	}
	if got[0].ID != "run-2" || got[1].ID != "run-4" {
		t.Fatalf("failed runs = %+v", got)
	}
}

func TestFilterFlowRunsLimitKeepsNewestRuns(t *testing.T) {
	runs := []aiflow.Run{
		{ID: "run-1", Status: "done"},
		{ID: "run-2", Status: "failed"},
		{ID: "run-3", Status: "running"},
	}

	got := filterFlowRuns(runs, flowRunOptions{Limit: 2})
	if len(got) != 2 {
		t.Fatalf("filterFlowRuns returned %d runs, want 2: %+v", len(got), got)
	}
	if got[0].ID != "run-2" || got[1].ID != "run-3" {
		t.Fatalf("limited runs = %+v", got)
	}
}

func TestFilterFlowRunsCombinesPendingStatusAndLimit(t *testing.T) {
	runs := []aiflow.Run{
		{ID: "run-1", Status: "failed"},
		{ID: "run-2", Status: "done"},
		{ID: "run-3", Status: "failed"},
		{ID: "run-4", Status: "running"},
		{ID: "run-5", Status: "failed"},
	}

	got := filterFlowRuns(runs, flowRunOptions{Pending: true, Status: "failed", Limit: 2})
	if len(got) != 2 {
		t.Fatalf("filterFlowRuns returned %d runs, want 2: %+v", len(got), got)
	}
	if got[0].ID != "run-3" || got[1].ID != "run-5" {
		t.Fatalf("filtered runs = %+v", got)
	}
}
