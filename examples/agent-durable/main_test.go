package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDurableAgentExampleResumesWithoutReplayingTool(t *testing.T) {
	out := captureStdout(t, main)
	if !strings.Contains(out, "simulated process interruption after checkpointed tool call") {
		t.Fatalf("example output %q did not show the initial interrupted run", out)
	}
	if !strings.Contains(out, "resumed reply: sku-123 is reserved; no duplicate reservation was made") {
		t.Fatalf("example output %q did not show the resumed response", out)
	}
	if !strings.Contains(out, "tool executions: 1") {
		t.Fatalf("example output %q did not prove the tool was not replayed", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stdout = old
	<-done
	_ = r.Close()
	return buf.String()
}
