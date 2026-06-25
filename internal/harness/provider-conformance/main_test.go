package main

import (
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
