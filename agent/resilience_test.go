package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
)

func TestAskCancellationAbortsPromptly(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("cancel"), ModelCallTimeout(time.Second), ModelRetry(3, time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := a.Ask(ctx, "stop")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Ask error = %v, want context canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Ask took %s after cancellation, want prompt abort", elapsed)
	}
}

func TestAskRetriesTransientErrorsThenSucceeds(t *testing.T) {
	attempts := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, context.DeadlineExceeded
		}
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("retry-success"), ModelRetry(3, time.Millisecond))
	resp, err := a.Ask(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	if resp.Reply != "ok" {
		t.Fatalf("reply = %q, want ok", resp.Reply)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestAskRetriesTransientErrorsThenSurfacesStructuredError(t *testing.T) {
	attempts := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		return nil, context.DeadlineExceeded
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("retry-fail"), ModelRetry(2, time.Millisecond))
	_, err := a.Ask(context.Background(), "hello")
	var retryErr *ai.RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("Ask error = %T %v, want *ai.RetryError", err, err)
	}
	if retryErr.Attempts != 2 {
		t.Fatalf("retry attempts = %d, want 2", retryErr.Attempts)
	}
	if attempts != 2 {
		t.Fatalf("model attempts = %d, want 2", attempts)
	}
}
