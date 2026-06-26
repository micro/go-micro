package ai

import (
	"context"
	"errors"
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
	attempts := 0
	model := retryModel{generate: func(ctx context.Context, _ *Request, _ ...GenerateOption) (*Response, error) {
		attempts++
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
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}
