package flow

import (
	"context"
	"strings"
	"testing"
)

func TestVerifyPassesFirstTry(t *testing.T) {
	attempts := 0
	step := Verify(func(_ context.Context, in State) (State, error) {
		attempts++
		in.Data = []byte(`{"answer":"ok"}`)
		return in, nil
	}, func(context.Context, State) (bool, string, error) {
		return true, "looks good", nil
	}, VerifyMaxAttempts(3))

	out, err := step(context.Background(), State{})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("body attempts = %d, want 1", attempts)
	}
	var got map[string]any
	if err := out.Scan(&got); err != nil {
		t.Fatalf("scan output: %v", err)
	}
	if got["verification_passed"] != true {
		t.Fatalf("verification_passed = %v, want true", got["verification_passed"])
	}
}

func TestVerifyRetriesWithFeedback(t *testing.T) {
	attempts := 0
	var secondInput map[string]string
	step := Verify(func(_ context.Context, in State) (State, error) {
		attempts++
		if attempts == 2 {
			if err := in.Scan(&secondInput); err != nil {
				t.Fatalf("scan second input: %v", err)
			}
		}
		in.Data = []byte(`{"answer":"draft"}`)
		return in, nil
	}, func(_ context.Context, _ State) (bool, string, error) {
		return attempts >= 2, "include citations", nil
	}, VerifyMaxAttempts(3))

	out, err := step(context.Background(), State{Data: []byte(`{"topic":"agents"}`)})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("body attempts = %d, want 2", attempts)
	}
	if secondInput["feedback"] != "include citations" {
		t.Fatalf("feedback = %q, want include citations", secondInput["feedback"])
	}
	if !strings.Contains(out.String(), `"verification_passed":true`) {
		t.Fatalf("output missing successful verification annotation: %s", out.String())
	}
}

func TestVerifyExhaustsAttemptsReturnsLastOutput(t *testing.T) {
	attempts := 0
	step := Verify(func(_ context.Context, in State) (State, error) {
		attempts++
		in.Data = []byte(`{"answer":"still wrong"}`)
		return in, nil
	}, func(context.Context, State) (bool, string, error) {
		return false, "try again", nil
	}, VerifyMaxAttempts(2))

	out, err := step(context.Background(), State{})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("body attempts = %d, want 2", attempts)
	}
	var got map[string]any
	if err := out.Scan(&got); err != nil {
		t.Fatalf("scan output: %v", err)
	}
	if got["verification_passed"] != false {
		t.Fatalf("verification_passed = %v, want false", got["verification_passed"])
	}
	if got["verification_feedback"] != "try again" {
		t.Fatalf("verification_feedback = %v, want try again", got["verification_feedback"])
	}
	if got["verification_attempts"] != float64(2) {
		t.Fatalf("verification_attempts = %v, want 2", got["verification_attempts"])
	}
}
