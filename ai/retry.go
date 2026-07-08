package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StatusCoder is implemented by provider errors that expose an HTTP-like status code.
type StatusCoder interface {
	StatusCode() int
}

// RetryAfterCoder is implemented by provider errors that expose a server
// supplied retry delay, such as HTTP Retry-After on a 429/503 response.
type RetryAfterCoder interface {
	RetryAfter() time.Duration
}

// HTTPError describes a failed provider HTTP response while preserving the
// status code and Retry-After signal for retry classifiers.
type HTTPError struct {
	Status string
	Code   int
	Body   string
	Header http.Header
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("API error (%s): %s", e.Status, e.Body)
}

func (e *HTTPError) StatusCode() int {
	if e == nil {
		return 0
	}
	return e.Code
}

func (e *HTTPError) RetryAfter() time.Duration {
	if e == nil {
		return 0
	}
	return parseRetryAfter(e.Header.Get("Retry-After"), time.Now())
}

func NewHTTPError(resp *http.Response, body []byte) error {
	if resp == nil {
		return errors.New("API error: nil response")
	}
	return &HTTPError{Status: resp.Status, Code: resp.StatusCode, Body: string(body), Header: resp.Header.Clone()}
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	if delay := when.Sub(now); delay > 0 {
		return delay
	}
	return 0
}

// ErrorKind classifies provider-boundary failures into stable buckets callers
// can inspect without parsing provider-specific error strings.
type ErrorKind string

const (
	ErrorKindUnknown     ErrorKind = "unknown"
	ErrorKindCanceled    ErrorKind = "canceled"
	ErrorKindTimeout     ErrorKind = "timeout"
	ErrorKindRateLimited ErrorKind = "rate_limited"
	ErrorKindUnavailable ErrorKind = "unavailable"
	ErrorKindProvider    ErrorKind = "provider"
)

// ClassifiedError is implemented by errors that expose a stable ErrorKind.
type ClassifiedError interface {
	ErrorKind() ErrorKind
}

// RetryError is returned when Generate is retried and still fails.
type RetryError struct {
	Attempts int
	Kind     ErrorKind
	Err      error
}

func (e *RetryError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("ai generate failed after %d attempt(s) (%s): %v", e.Attempts, e.ErrorKind(), e.Err)
}

func (e *RetryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *RetryError) ErrorKind() ErrorKind {
	if e == nil || e.Kind == "" {
		return ErrorKindUnknown
	}
	return e.Kind
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
		if info, ok := RunInfoFrom(callCtx); ok {
			info.Attempt = attempt
			info.MaxAttempts = policy.MaxAttempts
			callCtx = WithRunInfo(callCtx, info)
		}
		resp, err := m.Generate(callCtx, req, opts...)
		cancel()

		// Caller cancellation/deadline always wins and is not retried, even if
		// a provider or tool loop swallowed the canceled tool result and returned
		// a final response. This keeps agent runs from appearing successful after
		// their controlling context was abandoned.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if err == nil {
			return resp, nil
		}
		last = err

		transient := IsTransientError(err)
		if attempt == policy.MaxAttempts || !transient {
			if attempt > 1 || transient {
				return nil, &RetryError{Attempts: attempt, Kind: ClassifyError(err), Err: err}
			}
			return nil, err
		}

		// Always back off between retries — exponential and capped — so an
		// opt-in retry can never become a tight loop hammering the provider,
		// even if Backoff was left at zero.
		backoff := retryBackoff(err, attempt, policy.Backoff)
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
	return nil, &RetryError{Attempts: policy.MaxAttempts, Kind: ClassifyError(last), Err: last}
}

func retryBackoff(err error, attempt int, base time.Duration) time.Duration {
	backoff := base
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}
	if shift := attempt - 1; shift > 0 {
		backoff <<= shift
	}
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	var retryAfter RetryAfterCoder
	if errors.As(err, &retryAfter) {
		if delay := retryAfter.RetryAfter(); delay > backoff {
			backoff = delay
		}
	}
	if backoff > 30*time.Second {
		return 30 * time.Second
	}
	return backoff
}

// ClassifyError maps provider and context failures to stable operational kinds.
func ClassifyError(err error) ErrorKind {
	if err == nil {
		return ""
	}
	var classified ClassifiedError
	if errors.As(err, &classified) {
		if kind := classified.ErrorKind(); kind != "" {
			return kind
		}
	}
	if errors.Is(err, context.Canceled) {
		return ErrorKindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorKindTimeout
	}
	var sc StatusCoder
	if errors.As(err, &sc) {
		code := sc.StatusCode()
		switch {
		case code == 429:
			return ErrorKindRateLimited
		case code >= 500:
			return ErrorKindUnavailable
		case code > 0:
			return ErrorKindProvider
		}
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests"):
		return ErrorKindRateLimited
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return ErrorKindTimeout
	case strings.Contains(msg, "temporar") || strings.Contains(msg, "unavailable"):
		return ErrorKindUnavailable
	default:
		return ErrorKindUnknown
	}
}

// IsTransientError reports whether err is worth retrying at the provider boundary.
func IsTransientError(err error) bool {
	switch ClassifyError(err) {
	case ErrorKindTimeout, ErrorKindRateLimited, ErrorKindUnavailable:
		return true
	default:
		return false
	}
}
