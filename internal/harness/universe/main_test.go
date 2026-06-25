package main

import (
	"testing"
)

// TestUniverseHarnessContract makes the 0→hero harness part of the ordinary
// Go test contract. The harness boots real services, a durable workflow, an
// agent, scoped state, and the A2A gateway with only the LLM mocked; running it
// here prevents the full services → agents → workflows lifecycle from silently
// drifting while developers rely on `go test ./...`.
func TestUniverseHarnessContract(t *testing.T) {
	if testing.Short() {
		t.Skip("universe harness boots an end-to-end system; skipped with -short")
	}

	if code := runUniverse("mock"); code != 0 {
		t.Fatalf("universe harness exited with code %d", code)
	}
}
