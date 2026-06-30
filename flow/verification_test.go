package flow

import (
	"context"
	"errors"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
)

func TestFlowStepVerificationRetriesWithFeedback(t *testing.T) {
	var attempts int
	var feedback []string
	step := Step{
		Name:  "draft",
		Retry: 1,
		Run: func(ctx context.Context, in State) (State, error) {
			attempts++
			info, ok := ai.RunInfoFrom(ctx)
			if !ok {
				t.Fatal("RunInfo missing from verified step")
			}
			feedback = append(feedback, info.VerificationFeedback)
			if info.VerificationFeedback == "add evidence" {
				in.Data = []byte("answer with evidence")
			} else {
				in.Data = []byte("answer")
			}
			return in, nil
		},
		Verify: func(ctx context.Context, out State) (Verification, error) {
			if out.String() == "answer with evidence" {
				return Verification{Passed: true, Feedback: "meets rubric"}, nil
			}
			return Verification{Feedback: "add evidence"}, nil
		},
	}

	cp := StoreCheckpoint(store.NewMemoryStore(), "verified")
	f := New("verified", WithCheckpoint(cp), Steps(step))
	if err := f.Execute(context.Background(), "question"); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if len(feedback) != 2 || feedback[0] != "" || feedback[1] != "add evidence" {
		t.Fatalf("feedback = %#v, want empty then verifier feedback", feedback)
	}
	runs, err := cp.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(runs))
	}
	stepRecord := runs[0].Steps[0]
	if stepRecord.Status != "done" || stepRecord.Attempts != 2 || stepRecord.VerificationStatus != "passed" || stepRecord.VerificationNote != "meets rubric" {
		t.Fatalf("step record = %#v", stepRecord)
	}
}

func TestFlowStepVerificationFailureIsCheckpointed(t *testing.T) {
	step := Step{
		Name: "grade",
		Run: func(ctx context.Context, in State) (State, error) {
			in.Data = []byte("bad")
			return in, nil
		},
		Verify: func(ctx context.Context, out State) (Verification, error) {
			return Verification{Feedback: "missing citation"}, nil
		},
	}

	cp := StoreCheckpoint(store.NewMemoryStore(), "verified-fail")
	f := New("verified-fail", WithCheckpoint(cp), Steps(step))
	err := f.Execute(context.Background(), "question")
	if err == nil {
		t.Fatal("Execute succeeded, want verification failure")
	}
	var verr *VerificationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T %v, want VerificationError", err, err)
	}
	if verr.Feedback != "missing citation" {
		t.Fatalf("feedback = %q, want missing citation", verr.Feedback)
	}
	runs, listErr := cp.List(context.Background())
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(runs))
	}
	stepRecord := runs[0].Steps[0]
	if runs[0].Status != "failed" || stepRecord.VerificationStatus != "failed" || stepRecord.VerificationNote != "missing citation" {
		t.Fatalf("run = %#v step = %#v", runs[0], stepRecord)
	}
}
