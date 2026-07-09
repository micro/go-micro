package ai

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type retryModel struct {
	generate func(context.Context, *Request, ...GenerateOption) (*Response, error)
}

func (m retryModel) Init(...Option) error { return nil }
func (m retryModel) Options() Options     { return Options{} }
func (m retryModel) Generate(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error) {
	return m.generate(ctx, req, opts...)
}
func (m retryModel) Stream(context.Context, *Request, ...GenerateOption) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
func (m retryModel) String() string { return "retry-test" }

func TestGenerateWithRetryRetriesTransientErrors(t *testing.T) {
	attempts := 0
	model := retryModel{generate: func(context.Context, *Request, ...GenerateOption) (*Response, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("temporary provider outage")
		}
		return &Response{Reply: "ok"}, nil
	}}

	resp, err := GenerateWithRetry(context.Background(), model, &Request{Prompt: "hi"}, GeneratePolicy{
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("GenerateWithRetry returned error: %v", err)
	}
	if resp.Reply != "ok" {
		t.Fatalf("response reply = %q, want ok", resp.Reply)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestGenerateWithRetryDoesNotRetryCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	model := retryModel{generate: func(context.Context, *Request, ...GenerateOption) (*Response, error) {
		attempts++
		cancel()
		return nil, errors.New("temporary provider outage")
	}}

	_, err := GenerateWithRetry(ctx, model, &Request{Prompt: "hi"}, GeneratePolicy{
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestGenerateWithRetryHonorsPerAttemptTimeout(t *testing.T) {
	var attempts atomic.Int32
	model := retryModel{generate: func(ctx context.Context, _ *Request, _ ...GenerateOption) (*Response, error) {
		attempts.Add(1)
		<-ctx.Done()
		return nil, ctx.Err()
	}}

	_, err := GenerateWithRetry(context.Background(), model, &Request{Prompt: "hi"}, GeneratePolicy{
		Timeout:     time.Millisecond,
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
	})
	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("error = %T %[1]v, want RetryError", err)
	}
	if retryErr.Attempts != 2 {
		t.Fatalf("retry attempts = %d, want 2", retryErr.Attempts)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

func TestGenerateWithRetryAddsAttemptMetadataToRunInfo(t *testing.T) {
	var got []RunInfo
	model := retryModel{generate: func(ctx context.Context, _ *Request, _ ...GenerateOption) (*Response, error) {
		info, ok := RunInfoFrom(ctx)
		if !ok {
			t.Fatal("RunInfo missing from attempt context")
		}
		got = append(got, info)
		if info.Attempt == 1 {
			return nil, errors.New("temporary provider outage")
		}
		return &Response{Reply: "ok"}, nil
	}}

	ctx := WithRunInfo(context.Background(), RunInfo{RunID: "run-1", Agent: "worker"})
	_, err := GenerateWithRetry(ctx, model, &Request{Prompt: "hi"}, GeneratePolicy{
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("GenerateWithRetry returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("attempt contexts = %d, want 2", len(got))
	}
	for i, info := range got {
		wantAttempt := i + 1
		if info.Attempt != wantAttempt {
			t.Fatalf("attempt %d RunInfo.Attempt = %d, want %d", i, info.Attempt, wantAttempt)
		}
		if info.MaxAttempts != 2 {
			t.Fatalf("attempt %d RunInfo.MaxAttempts = %d, want 2", i, info.MaxAttempts)
		}
		if info.RunID != "run-1" || info.Agent != "worker" {
			t.Fatalf("attempt %d RunInfo identity = (%q, %q), want (run-1, worker)", i, info.RunID, info.Agent)
		}
	}
}

func TestGenerateWithRetryReturnsWhenProviderIgnoresTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	model := retryModel{generate: func(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error) {
		close(started)
		<-release
		return &Response{Reply: "late"}, nil
	}}
	defer close(release)

	start := time.Now()
	_, err := GenerateWithRetry(context.Background(), model, &Request{Prompt: "hi"}, GeneratePolicy{
		Timeout:     10 * time.Millisecond,
		MaxAttempts: 1,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("GenerateWithRetry error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("GenerateWithRetry took %s after deadline, want prompt return", elapsed)
	}
	select {
	case <-started:
	default:
		t.Fatal("provider was not called")
	}
}

type statusErr int

func (e statusErr) Error() string   { return "provider status" }
func (e statusErr) StatusCode() int { return int(e) }

type retryAfterErr struct {
	delay time.Duration
}

func (e retryAfterErr) Error() string             { return "rate limit exceeded" }
func (e retryAfterErr) StatusCode() int           { return 429 }
func (e retryAfterErr) RetryAfter() time.Duration { return e.delay }

func TestClassifyErrorDistinguishesOperationalOutcomes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorKind
	}{
		{name: "canceled", err: context.Canceled, want: ErrorKindCanceled},
		{name: "timeout", err: context.DeadlineExceeded, want: ErrorKindTimeout},
		{name: "rate limit status", err: statusErr(429), want: ErrorKindRateLimited},
		{name: "unavailable status", err: statusErr(503), want: ErrorKindUnavailable},
		{name: "provider status", err: statusErr(400), want: ErrorKindProvider},
		{name: "rate limit text", err: errors.New("rate limit exceeded"), want: ErrorKindRateLimited},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyError(tt.err); got != tt.want {
				t.Fatalf("ClassifyError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateWithRetryExposesRetryErrorKind(t *testing.T) {
	model := retryModel{generate: func(context.Context, *Request, ...GenerateOption) (*Response, error) {
		return nil, statusErr(429)
	}}

	_, err := GenerateWithRetry(context.Background(), model, &Request{Prompt: "hi"}, GeneratePolicy{
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
	})
	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("error = %T %[1]v, want RetryError", err)
	}
	if retryErr.ErrorKind() != ErrorKindRateLimited {
		t.Fatalf("retry kind = %q, want %q", retryErr.ErrorKind(), ErrorKindRateLimited)
	}
	if !errors.Is(err, statusErr(429)) {
		t.Fatalf("retry error does not unwrap provider status: %v", err)
	}
}

func TestGenerateWithRetryHonorsRetryAfterWhenLongerThanBackoff(t *testing.T) {
	attempts := 0
	model := retryModel{generate: func(context.Context, *Request, ...GenerateOption) (*Response, error) {
		attempts++
		if attempts == 1 {
			return nil, retryAfterErr{delay: 25 * time.Millisecond}
		}
		return &Response{Reply: "ok"}, nil
	}}

	start := time.Now()
	resp, err := GenerateWithRetry(context.Background(), model, &Request{Prompt: "hi"}, GeneratePolicy{
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("GenerateWithRetry returned error: %v", err)
	}
	if resp.Reply != "ok" {
		t.Fatalf("reply = %q, want ok", resp.Reply)
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("retry delay = %s, want RetryAfter delay to dominate base backoff", elapsed)
	}
}

func TestGenerateWithRetryCapsRetryAfter(t *testing.T) {
	if got := retryBackoff(retryAfterErr{delay: time.Minute}, 1, time.Millisecond); got != 30*time.Second {
		t.Fatalf("retryBackoff() = %s, want 30s cap", got)
	}
}

func TestHTTPErrorExposesStatusAndRetryAfter(t *testing.T) {
	resp := &http.Response{
		Status:     "429 Too Many Requests",
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"2"}},
	}
	err := NewHTTPError(resp, []byte("slow down"))

	if got := ClassifyError(err); got != ErrorKindRateLimited {
		t.Fatalf("ClassifyError() = %q, want %q", got, ErrorKindRateLimited)
	}
	var retryAfter RetryAfterCoder
	if !errors.As(err, &retryAfter) {
		t.Fatalf("NewHTTPError does not expose RetryAfterCoder")
	}
	if got := retryAfter.RetryAfter(); got != 2*time.Second {
		t.Fatalf("RetryAfter() = %s, want 2s", got)
	}
}
