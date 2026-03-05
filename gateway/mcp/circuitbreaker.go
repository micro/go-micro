package mcp

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerConfig configures circuit breaking for the MCP gateway.
// When a downstream service fails repeatedly, the circuit opens and
// subsequent calls are rejected immediately until the service recovers.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of consecutive failures before the circuit opens.
	// Default: 5
	MaxFailures int

	// Timeout is how long the circuit stays open before allowing a probe request.
	// Default: 30s
	Timeout time.Duration

	// MaxHalfOpen is the number of probe requests allowed in the half-open state.
	// If they all succeed, the circuit closes. If any fail, it re-opens.
	// Default: 1
	MaxHalfOpen int
}

// circuitState represents the state of a circuit breaker.
type circuitState int

const (
	circuitClosed   circuitState = iota // healthy, requests flow through
	circuitOpen                         // tripped, requests are rejected
	circuitHalfOpen                     // testing recovery with limited requests
)

func (s circuitState) String() string {
	switch s {
	case circuitClosed:
		return "closed"
	case circuitOpen:
		return "open"
	case circuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// circuitBreaker tracks failure state for a single tool/service endpoint.
type circuitBreaker struct {
	mu           sync.Mutex
	state        circuitState
	failures     int
	maxFailures  int
	timeout      time.Duration
	maxHalfOpen  int
	halfOpenUsed int
	lastFailure  time.Time
}

func newCircuitBreaker(cfg CircuitBreakerConfig) *circuitBreaker {
	maxFailures := cfg.MaxFailures
	if maxFailures <= 0 {
		maxFailures = 5
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	maxHalfOpen := cfg.MaxHalfOpen
	if maxHalfOpen <= 0 {
		maxHalfOpen = 1
	}
	return &circuitBreaker{
		state:       circuitClosed,
		maxFailures: maxFailures,
		timeout:     timeout,
		maxHalfOpen: maxHalfOpen,
	}
}

// Allow checks whether a request should be allowed through.
// Returns nil if allowed, error if the circuit is open.
func (cb *circuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitClosed:
		return nil
	case circuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = circuitHalfOpen
			cb.halfOpenUsed = 0
			return nil
		}
		return fmt.Errorf("circuit breaker open (consecutive failures: %d)", cb.failures)
	case circuitHalfOpen:
		if cb.halfOpenUsed < cb.maxHalfOpen {
			cb.halfOpenUsed++
			return nil
		}
		return fmt.Errorf("circuit breaker half-open (probe limit reached)")
	}
	return nil
}

// RecordSuccess records a successful call. If half-open, closes the circuit.
func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = circuitClosed
}

// RecordFailure records a failed call. May trip the circuit open.
func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case circuitClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = circuitOpen
		}
	case circuitHalfOpen:
		// Probe failed, re-open
		cb.state = circuitOpen
	}
}

// State returns the current circuit state.
func (cb *circuitBreaker) State() circuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check for automatic transition from open -> half-open
	if cb.state == circuitOpen && time.Since(cb.lastFailure) > cb.timeout {
		cb.state = circuitHalfOpen
		cb.halfOpenUsed = 0
	}
	return cb.state
}
