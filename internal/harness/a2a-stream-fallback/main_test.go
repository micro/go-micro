package main

import (
	"strings"
	"testing"
)

func TestReadSSESummaryUsesCompletedTaskInvariants(t *testing.T) {
	summary, err := readSSESummary(strings.NewReader("data: {\"jsonrpc\":\"2.0\",\"result\":{\"status\":{\"state\":\"working\"}}}\n\n" +
		"data: {\"jsonrpc\":\"2.0\",\"result\":{\"status\":{\"state\":\"completed\"},\"artifacts\":[{\"parts\":[{\"kind\":\"text\",\"text\":\"provider-specific answer\"}]}]}}\n\n"))
	if err != nil {
		t.Fatalf("readSSESummary() error = %v", err)
	}
	if summary.State != "completed" {
		t.Fatalf("State = %q, want completed", summary.State)
	}
	if !summary.HasArtifactText {
		t.Fatal("HasArtifactText = false, want true")
	}
	if strings.Contains(summary.Payload, "a2a-fallback-ok") {
		t.Fatalf("test fixture should not rely on marker text: %s", summary.Payload)
	}
}

func TestReadSSESummaryRejectsNonJSONData(t *testing.T) {
	_, err := readSSESummary(strings.NewReader("data: not-json\n\n"))
	if err == nil {
		t.Fatal("readSSESummary() error = nil, want non-JSON error")
	}
}
