package otel

import (
	"sync/atomic"
	"testing"
)

func TestInitNoopWithoutBackend(t *testing.T) {
	// Reset for the test: no backend registered => Init/Shutdown must no-op.
	build, flush = nil, nil
	Init()
	Shutdown()
}

func TestRegisterWiresInitAndShutdown(t *testing.T) {
	var initCalls, shutdownCalls atomic.Int32
	Register(func() { initCalls.Add(1) }, func() { shutdownCalls.Add(1) })

	Init()
	Init()
	if got := initCalls.Load(); got != 2 {
		t.Fatalf("Init() = %d calls, want 2", got)
	}
	Shutdown()
	if got := shutdownCalls.Load(); got != 1 {
		t.Fatalf("Shutdown() = %d calls, want 1", got)
	}

	// Restore clean state for any later tests.
	build, flush = nil, nil
}
