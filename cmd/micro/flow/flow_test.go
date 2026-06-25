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
