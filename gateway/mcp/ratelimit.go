package mcp

import (
	"sync"
	"time"
)

// rateLimiter implements a simple token-bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	rate     float64   // tokens per second
	burst    int       // max tokens
	tokens   float64   // current token count
	lastTime time.Time // last refill time
}

// newRateLimiter creates a rate limiter that allows rate requests/sec with
// the given burst size. If burst is less than 1 it defaults to 1.
func newRateLimiter(rate float64, burst int) *rateLimiter {
	if burst < 1 {
		burst = 1
	}
	return &rateLimiter{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

// Allow reports whether a single event may happen now.
func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastTime).Seconds()
	r.lastTime = now

	// Refill tokens based on elapsed time
	r.tokens += elapsed * r.rate
	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}

	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}
