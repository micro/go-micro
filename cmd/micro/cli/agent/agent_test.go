package agent

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
)

func TestWriteRunIndexJSON(t *testing.T) {
	runs := []goagent.RunSummary{{
		RunID:     "run-1",
		Agent:     "runner",
		StartedAt: time.Unix(0, 1),
		UpdatedAt: time.Unix(0, 2),
		Events:    2,
		LastKind:  "tool",
		TraceID:   "1234567890abcdef",
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
	}}

	var human bytes.Buffer
	if err := writeRunHistory(&human, "runner", "run-1", events, false); err != nil {
		t.Fatal(err)
	}
	line := human.String()
	for _, want := range []string{"12:34:56.007 tool", "probe", "oteltest/unit-model", "42ms", "tokens=5", "trace=1234567890ab"} {
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
