package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go-micro.dev/v6/flow"
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

func TestNotifyStepCompletesAfterObservedSideEffectTimeout(t *testing.T) {
	ntf := new(Notify)
	before := atomic.LoadInt64(&ntf.sent)
	go func() {
		time.Sleep(30 * time.Millisecond)
		var rsp SendResponse
		if err := ntf.Send(context.Background(), &SendRequest{
			To:      "buyer@acme.com",
			Message: "Your order is confirmed.",
		}, &rsp); err != nil {
			t.Errorf("send notification: %v", err)
		}
	}()

	out, err := completeNotifyOnObservedSideEffect(
		context.Background(),
		flow.State{Data: []byte(`{"order":"order-1"}`)},
		ntf,
		before,
		time.Second,
		errors.New("client observed timeout"),
	)
	if err != nil {
		t.Fatalf("notify completion returned error: %v", err)
	}
	if got := out.String(); got != "Buyer notified." {
		t.Fatalf("result = %q, want Buyer notified.", got)
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("notifications sent = %d, want 1", got)
	}

	var rsp SendResponse
	if err := ntf.Send(context.Background(), &SendRequest{
		To:      "buyer@acme.com",
		Message: "Your order is confirmed.",
	}, &rsp); err != nil {
		t.Fatalf("duplicate send: %v", err)
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("notifications sent after duplicate = %d, want 1", got)
	}
}

func TestNotifyStepWaitsForObservedSideEffectAfterCanceledDispatchContext(t *testing.T) {
	ntf := new(Notify)
	before := atomic.LoadInt64(&ntf.sent)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	go func() {
		time.Sleep(30 * time.Millisecond)
		var rsp SendResponse
		if err := ntf.Send(context.Background(), &SendRequest{
			To:      "buyer@acme.com",
			Message: "Your order is confirmed.",
		}, &rsp); err != nil {
			t.Errorf("send notification: %v", err)
		}
	}()

	out, err := completeNotifyOnObservedSideEffect(
		ctx,
		flow.State{Data: []byte(`{"order":"order-1"}`)},
		ntf,
		before,
		time.Second,
		errors.New("client observed timeout"),
	)
	if err != nil {
		t.Fatalf("notify completion returned error: %v", err)
	}
	if got := out.String(); got != "Buyer notified." {
		t.Fatalf("result = %q, want Buyer notified.", got)
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("notifications sent = %d, want 1", got)
	}
}

func TestNotifyStepRejectsClaimedCompletionWithoutSideEffect(t *testing.T) {
	ntf := new(Notify)
	before := atomic.LoadInt64(&ntf.sent)

	_, err := completeNotifyOnObservedSideEffect(
		context.Background(),
		flow.State{Data: []byte(`claimed success`)},
		ntf,
		before,
		25*time.Millisecond,
		nil,
	)
	if err == nil {
		t.Fatal("notify completion returned nil, want missing buyer notification error")
	}
	want := `concierge completed without notifying buyer: notify count stayed at 0; expected recipient buyer@acme.com, buyer, or buyer-of-order-<id>; no rejected notify call observed`
	if got := err.Error(); got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNotifySuppressesEquivalentConfirmationMessages(t *testing.T) {
	ntf := new(Notify)
	ctx := context.Background()

	for _, req := range []*SendRequest{
		{To: "buyer@acme.com", Message: "Your order order-1 has been confirmed."},
		{To: "buyer@acme.com", Message: "order-1 confirmed"},
	} {
		var rsp SendResponse
		if err := ntf.Send(ctx, req, &rsp); err != nil {
			t.Fatalf("send notification %q: %v", req.Message, err)
		}
		if !rsp.Sent {
			t.Fatalf("send notification %q did not report sent", req.Message)
		}
	}

	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("equivalent confirmation notifications sent = %d, want 1", got)
	}
}

func TestNotifyAcceptsBuyerAlias(t *testing.T) {
	ntf := new(Notify)
	ctx := context.Background()

	var rsp SendResponse
	if err := ntf.Send(ctx, &SendRequest{
		To:      "buyer",
		Message: "Your order order-1 has been confirmed.",
	}, &rsp); err != nil {
		t.Fatalf("send buyer alias notification: %v", err)
	}
	if !rsp.Sent {
		t.Fatal("buyer alias notification did not report sent")
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("buyer alias notifications sent = %d, want 1", got)
	}

	if err := ntf.Send(ctx, &SendRequest{
		To:      "buyer@acme.com",
		Message: "order-1 confirmed",
	}, &rsp); err != nil {
		t.Fatalf("send canonical buyer notification: %v", err)
	}
	if !rsp.Sent {
		t.Fatal("canonical buyer notification did not report sent")
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("alias/canonical confirmation notifications sent = %d, want 1", got)
	}
}

func TestNotifyIgnoresNonBuyerRecipients(t *testing.T) {
	ntf := new(Notify)
	ctx := context.Background()

	var rsp SendResponse
	if err := ntf.Send(ctx, &SendRequest{
		To:      "order-1",
		Message: "order-1 confirmed",
	}, &rsp); err != nil {
		t.Fatalf("send non-buyer notification: %v", err)
	}
	if rsp.Sent {
		t.Fatal("non-buyer notification reported sent")
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 0 {
		t.Fatalf("non-buyer notifications sent = %d, want 0", got)
	}

	if err := ntf.Send(ctx, &SendRequest{
		To:      "buyer@acme.com",
		Message: "Your order order-1 has been confirmed.",
	}, &rsp); err != nil {
		t.Fatalf("send buyer notification: %v", err)
	}
	if !rsp.Sent {
		t.Fatal("buyer notification did not report sent")
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("buyer notifications sent = %d, want 1", got)
	}
}

func TestNotifyAcceptsOrderScopedBuyerRecipient(t *testing.T) {
	ntf := new(Notify)
	ctx := context.Background()

	var rsp SendResponse
	if err := ntf.Send(ctx, &SendRequest{
		To:      "buyer-of-order-1",
		Message: "order-1 confirmed",
	}, &rsp); err != nil {
		t.Fatalf("send order-scoped buyer notification: %v", err)
	}
	if !rsp.Sent {
		t.Fatal("order-scoped buyer notification did not report sent")
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("order-scoped buyer notifications sent = %d, want 1", got)
	}

	if err := ntf.Send(ctx, &SendRequest{
		To:      "non-buyer",
		Message: "order-1 confirmed",
	}, &rsp); err != nil {
		t.Fatalf("send hyphenated non-buyer notification: %v", err)
	}
	if rsp.Sent {
		t.Fatal("hyphenated non-buyer notification reported sent")
	}
	if got := atomic.LoadInt64(&ntf.sent); got != 1 {
		t.Fatalf("notifications sent after hyphenated non-buyer = %d, want 1", got)
	}
}

func TestNotifyStepReportsRejectedRecipientDiagnostics(t *testing.T) {
	ntf := new(Notify)
	var rsp SendResponse
	if err := ntf.Send(context.Background(), &SendRequest{
		To:      "order-1",
		Message: "order-1 confirmed",
	}, &rsp); err != nil {
		t.Fatalf("send rejected notification: %v", err)
	}

	_, err := completeNotifyOnObservedSideEffect(
		context.Background(),
		flow.State{Data: []byte(`claimed success`)},
		ntf,
		0,
		25*time.Millisecond,
		nil,
	)
	if err == nil {
		t.Fatal("notify completion returned nil, want diagnostics")
	}
	want := `concierge completed without notifying buyer: notify count stayed at 0; expected recipient buyer@acme.com, buyer, or buyer-of-order-<id>; last notify args to="order-1" message="order-1 confirmed"`
	if got := err.Error(); got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
