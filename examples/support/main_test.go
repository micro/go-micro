package main

import (
	"os"
	"strings"
	"testing"
)

func TestRunSupportMockSmoke(t *testing.T) {
	if err := runSupport("mock"); err != nil {
		t.Fatalf("support example failed: %v", err)
	}
}

func TestZeroToHeroReadmeDocumentsLifecycle(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	doc := string(b)
	for _, want := range []string{
		"Scaffold services",
		"Run the harness",
		"Chat through an agent",
		"Inspect the workflow",
		"go test ./examples/support",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("README.md missing zero-to-hero step %q", want)
		}
	}
}
