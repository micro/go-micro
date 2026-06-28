package inspect

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	goagent "go-micro.dev/v6/agent"
	aiflow "go-micro.dev/v6/flow"
)

func TestWriteAgentInspectionIncludesActionableBreadcrumbs(t *testing.T) {
	runs := []goagent.RunSummary{{RunID: "run-1", Status: "error", Events: 4, LastKind: "tool", LastError: "boom", TraceID: "1234567890abcdef"}}
	var out bytes.Buffer
	if err := writeAgentInspection(&out, "support", runs, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"Agent \"support\" runs", "run-1", "status=error", "events=4", "last=tool", `error="boom"`, "trace=1234567890ab"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
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
