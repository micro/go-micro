package mcp

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedAllowsRequests(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{MaxFailures: 3, Timeout: time.Second})
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected closed circuit to allow, got: %v", err)
	}
	if cb.State() != circuitClosed {
		t.Fatalf("expected closed state, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{MaxFailures: 3, Timeout: time.Minute})

	// 2 failures: still closed
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != circuitClosed {
		t.Fatalf("expected closed after 2 failures, got %s", cb.State())
	}

	// 3rd failure: trips open
	cb.RecordFailure()
	if cb.State() != circuitOpen {
		t.Fatalf("expected open after 3 failures, got %s", cb.State())
	}

	// Requests should be rejected
	if err := cb.Allow(); err == nil {
		t.Fatal("expected open circuit to reject")
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{MaxFailures: 3, Timeout: time.Minute})

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // resets
	cb.RecordFailure()
	cb.RecordFailure()

	// Should still be closed (only 2 consecutive failures)
	if cb.State() != circuitClosed {
		t.Fatalf("expected closed after reset, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 1,
		Timeout:     50 * time.Millisecond,
		MaxHalfOpen: 1,
	})

	cb.RecordFailure()
	if cb.State() != circuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open
	if cb.State() != circuitHalfOpen {
		t.Fatalf("expected half-open after timeout, got %s", cb.State())
	}

	// One probe request should be allowed
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected half-open to allow probe, got: %v", err)
	}

	// Second should be rejected (maxHalfOpen=1, already used)
	if err := cb.Allow(); err == nil {
		t.Fatal("expected half-open to reject after max probes")
	}
}

func TestCircuitBreaker_HalfOpenSuccessCloses(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 1,
		Timeout:     50 * time.Millisecond,
	})

	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)

	// Allow probe
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected probe allowed: %v", err)
	}

	// Probe succeeds -> circuit closes
	cb.RecordSuccess()
	if cb.State() != circuitClosed {
		t.Fatalf("expected closed after successful probe, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 1,
		Timeout:     50 * time.Millisecond,
	})

	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)

	// Allow probe
	cb.Allow()

	// Probe fails -> circuit re-opens
	cb.RecordFailure()
	if cb.State() != circuitOpen {
		t.Fatalf("expected open after failed probe, got %s", cb.State())
	}
}

func TestCircuitBreaker_Defaults(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{})

	if cb.maxFailures != 5 {
		t.Fatalf("expected default maxFailures=5, got %d", cb.maxFailures)
	}
	if cb.timeout != 30*time.Second {
		t.Fatalf("expected default timeout=30s, got %s", cb.timeout)
	}
	if cb.maxHalfOpen != 1 {
		t.Fatalf("expected default maxHalfOpen=1, got %d", cb.maxHalfOpen)
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state circuitState
		want  string
	}{
		{circuitClosed, "closed"},
		{circuitOpen, "open"},
		{circuitHalfOpen, "half-open"},
		{circuitState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("state %d: got %q, want %q", tt.state, got, tt.want)
		}
	}
}
