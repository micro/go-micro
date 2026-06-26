package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestValidateSelectionAcceptsKnownProviderAndHarness(t *testing.T) {
	if err := validateSelection([]string{"mock"}, []string{"provider-conformance"}); err != nil {
		t.Fatalf("validateSelection returned error for known selection: %v", err)
	}
}

func TestValidateSelectionRejectsUnknownProvider(t *testing.T) {
	err := validateSelection([]string{"not-a-provider"}, []string{"provider-conformance"})
	if err == nil {
		t.Fatal("validateSelection returned nil for unknown provider")
	}
	if !strings.Contains(err.Error(), `unknown provider "not-a-provider"`) {
		t.Fatalf("validateSelection error = %q, want unknown provider message", err)
	}
}

func TestValidateSelectionRejectsUnsafeHarnessName(t *testing.T) {
	err := validateSelection([]string{"mock"}, []string{"../agent-flow"})
	if err == nil {
		t.Fatal("validateSelection returned nil for unsafe harness name")
	}
	if !strings.Contains(err.Error(), `invalid harness name "../agent-flow"`) {
		t.Fatalf("validateSelection error = %q, want invalid harness message", err)
	}
}

func TestCapabilityMatrixHasRegisteredProviders(t *testing.T) {
	rows := ai.CapabilityRows()
	if len(rows) == 0 {
		t.Fatal("CapabilityRows returned no providers")
	}

	var foundOpenAI bool
	for _, row := range rows {
		if row.Provider == "openai" {
			foundOpenAI = true
			if !row.Model || !row.Image || row.Video {
				t.Fatalf("openai capabilities = %#v, want model+image only", row.Capabilities)
			}
		}
	}
	if !foundOpenAI {
		t.Fatalf("CapabilityRows = %#v, want openai row", rows)
	}
}

func TestWriteCapabilityMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capabilities.md")
	rows := []ai.CapabilityRow{
		{Provider: "mock", Capabilities: ai.Capabilities{Model: true}},
		{Provider: "vision", Capabilities: ai.Capabilities{Image: true, Video: true}},
	}
	if err := writeCapabilityMarkdown(path, rows); err != nil {
		t.Fatalf("writeCapabilityMarkdown returned error: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read capabilities markdown: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		"| Provider | Model | Image | Video |",
		"| mock | ✅ | — | — |",
		"| vision | — | ✅ | ✅ |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("capabilities markdown = %q, want row %q", got, want)
		}
	}
}

func TestWriteSummaryJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.json")
	summary := conformanceSummary{
		Providers: []string{"mock"},
		Harnesses: []string{"provider-conformance"},
		Results: []conformanceResult{{
			Provider: "mock",
			Harness:  "provider-conformance",
			Status:   statusPassed,
		}},
		Passed: 1,
	}
	if err := writeSummaryJSON(path, summary); err != nil {
		t.Fatalf("writeSummaryJSON returned error: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if !strings.HasSuffix(string(b), "\n") {
		t.Fatalf("summary JSON should end with newline: %q", b)
	}

	var got conformanceSummary
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("summary JSON did not decode: %v", err)
	}
	if got.Passed != 1 || len(got.Results) != 1 || got.Results[0].Status != statusPassed {
		t.Fatalf("summary JSON decoded as %#v, want one passed result", got)
	}
}
