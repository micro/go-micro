package main

import (
	"strings"
	"testing"
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
