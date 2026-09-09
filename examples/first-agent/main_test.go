package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunFirstAgent(t *testing.T) {
	var out bytes.Buffer
	if err := runFirstAgentWithWriter(&out); err != nil {
		t.Fatalf("first-agent example failed: %v", err)
	}

	want := strings.TrimSpace(readExpectedTranscript(t))
	got := strings.TrimSpace(out.String())
	if got != want {
		t.Fatalf("first-agent transcript drifted from README.md\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestReadmeDocumentsNextBreadcrumbs(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(b)
	start := strings.Index(readme, "## Next chat, inspect, and debug breadcrumbs")
	if start < 0 {
		t.Fatal("README.md missing next chat, inspect, and debug breadcrumbs section")
	}
	section := readme[start:]
	for _, want := range []string{
		"micro run",
		"micro chat assistant --prompt \"Summarize my next steps\"",
		"micro inspect agent assistant",
		"micro agent doctor assistant",
		"no-secret-first-agent.md",
		"debugging-agents.md",
		"examples/support",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("README.md next breadcrumbs missing %q", want)
		}
	}
}

func readExpectedTranscript(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(b)
	const fence = "```text"
	start := strings.Index(readme, "Expected transcript:")
	if start < 0 {
		t.Fatal("README.md missing Expected transcript section")
	}
	fenceStart := strings.Index(readme[start:], fence)
	if fenceStart < 0 {
		t.Fatal("README.md missing transcript text fence")
	}
	start += fenceStart + len(fence)
	end := strings.Index(readme[start:], "```")
	if end < 0 {
		t.Fatal("README.md missing closing transcript fence")
	}
	return readme[start : start+end]
}
