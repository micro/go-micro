package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// StatusCoder is implemented by provider errors that expose an HTTP-like status code.
type StatusCoder interface {
	StatusCode() int
}

// RetryError is returned when Generate is retried and still fails.
type RetryError struct {
	Attempts int
	Err      error
}

func (e *RetryError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("ai generate failed after %d attempt(s): %v", e.Attempts, e.Err)
}

func (e *RetryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// GeneratePolicy controls timeout and retry behavior for a model call.
type GeneratePolicy struct {
	Timeout     time.Duration
	MaxAttempts int
	Backoff     time.Duration
}

// GenerateWithRetry calls m.Generate with per-attempt timeout and bounded retry.
func GenerateWithRetry(ctx context.Context, m Model, req *Request, policy GeneratePolicy, opts ...GenerateOption) (*Response, error) {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if m == nil {
		return nil, errors.New("ai model is nil")
	}

	var last error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		callCtx := ctx
		cancel := func() {}
		if policy.Timeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, policy.Timeout)
		}
		resp, err := m.Generate(callCtx, req, opts...)
		cancel()
		if err == nil {
			return resp, nil
		}
		last = err

		// Caller cancellation/deadline always wins and is not retried.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt == policy.MaxAttempts || !IsTransientError(err) {
			if attempt > 1 || IsTransientError(err) {
				return nil, &RetryError{Attempts: attempt, Err: err}
			}
			return nil, err
		}

		// Always back off between retries — exponential and capped — so an
		// opt-in retry can never become a tight loop hammering the provider,
		// even if Backoff was left at zero.
		backoff := policy.Backoff
		if backoff <= 0 {
			backoff = 200 * time.Millisecond
		}
		if shift := attempt - 1; shift > 0 {
			backoff <<= shift
		}
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			if !t.Stop() {
				<-t.C
			}
			return nil, ctx.Err()
		case <-t.C:
		}
	}
	return nil, &RetryError{Attempts: policy.MaxAttempts, Err: last}
}

// IsTransientError reports whether err is worth retrying at the provider boundary.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var sc StatusCoder
	if errors.As(err, &sc) {
		code := sc.StatusCode()
		return code == 429 || code >= 500
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") || strings.Contains(msg, "timeout") || strings.Contains(msg, "temporar")
}
