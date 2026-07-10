package main

import (
	"bytes"
	"io"
	"os"
	"regexp"
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

func TestZeroToHeroInspectTranscript(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runSupport("mock"); err != nil {
			t.Fatalf("support example failed: %v", err)
		}
	})
	got := stripANSI(out)

	for _, want := range []string{
		`> event: events.ticket.created {"customer":"alice@acme.com","id":"ticket-1","subject":"Can't log in"}`,
		`[customers] looked up Alice (pro plan)`,
		`[tickets] ticket-1 → priority=high status=in_progress`,
		`approval gate notify_NotifyService_Send(alice@acme.com) — approved`,
		`[notify] 📨 to=alice@acme.com: "Hi Alice — thanks for reaching out. We've bumped this to high priority and are on it."`,
		`support agent: Triaged ticket-1 for Alice and sent a reply.`,
		`inspect transcript:`,
		`micro inspect flow intake`,
		`flow: intake runs=1 latest.reply="Triaged ticket-1 for Alice and sent a reply."`,
		`micro agent history support`,
		`agent: support runs=1 latest.status=completed`,
		`✓ ticket triaged and the customer was replied to — triggered by an event`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("support transcript missing %q\n--- got ---\n%s", want, got)
		}
	}

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	for _, want := range []string{
		"Expected inspect transcript",
		"micro inspect flow intake",
		"micro agent history support",
		"agent: support runs=1 latest.status=completed",
	} {
		if !strings.Contains(string(readme), want) {
			t.Fatalf("README.md missing transcript contract %q", want)
		}
	}
}

func captureStdout(t *testing.T, fn func()) (out string) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()
	defer func() {
		_ = w.Close()
		os.Stdout = old
		<-done
		out = buf.String()
	}()

	fn()
	return out
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}
