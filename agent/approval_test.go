package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

type approvalIdentityKey struct{}

func TestDurableApprovalExecutesSavedCallAfterRestart(t *testing.T) {
	for _, approved := range []bool{true, false} {
		name := "approved"
		if !approved {
			name = "denied"
		}
		t.Run(name, func(t *testing.T) {
			st := store.NewMemoryStore()
			cp := flow.StoreCheckpoint(st, "approval")
			toolCalls, modelCalls, approvalCalls := 0, 0, 0
			ctx := context.WithValue(context.Background(), approvalIdentityKey{}, "reviewer")
			makeAgent := func() *agentImpl {
				return newTestAgent(Name("approval"), WithStore(st), WithCheckpoint(cp), WithApproval(func(ctx context.Context, call model.ToolCall) (ApprovalDecision, error) {
					approvalCalls++
					if ctx.Value(approvalIdentityKey{}) != "reviewer" {
						t.Error("identity lost")
					}
					if info, ok := model.RunInfoFrom(ctx); !ok || info.RunID == "" {
						t.Error("run identity lost")
					}
					return ApprovalDecision{Status: ApprovalPending, ID: "review-1", Reason: "human review required"}, nil
				}), WithTool("publish", "publish", nil, func(_ context.Context, input map[string]any) (string, error) {
					toolCalls++
					if input["body"] != "exact approved wording" {
						t.Errorf("changed input: %v", input)
					}
					return "published-once", nil
				}))
			}
			fakeGen = func(ctx context.Context, opts model.Options, req *model.Request) (*model.Response, error) {
				modelCalls++
				if modelCalls == 1 {
					call := model.ToolCall{ID: "original-call", Name: "publish", Input: map[string]any{"body": "exact approved wording"}}
					if result := opts.ToolHandler(ctx, call); result.Refused != model.RefusedApproval {
						t.Fatal("pending call not refused")
					}
					call.Input["body"] = "changed after approval"
					if result := opts.ToolHandler(ctx, call); result.Refused != model.RefusedApproval {
						t.Fatal("later call executed while paused")
					}
					return &model.Response{Reply: "waiting"}, nil
				}
				// The model deliberately does not regenerate the call. Its result must
				// already be available before this continuation starts.
				wantCalls := 0
				if approved {
					wantCalls = 1
				}
				if toolCalls != wantCalls {
					t.Errorf("tool was not resolved before model: %d", toolCalls)
				}
				found := false
				for _, message := range req.Messages {
					content, _ := message.Content.(string)
					if strings.Contains(content, "exact approved wording") && (strings.Contains(content, "published-once") || strings.Contains(content, "Approval denied")) {
						found = true
					}
				}
				if !found {
					t.Errorf("missing recorded outcome: %+v", req.Messages)
				}
				if approved && modelCalls == 2 {
					return nil, context.DeadlineExceeded
				}
				return &model.Response{Reply: "finished"}, nil
			}
			defer func() { fakeGen = nil }()
			first := makeAgent()
			_, err := first.Ask(ctx, "publish the announcement")
			var paused *PausedError
			if !errors.As(err, &paused) || paused.ApprovalID != "review-1" {
				t.Fatalf("pause=%v", err)
			}
			restarted := makeAgent()
			if _, err = Resume(ctx, restarted, paused.RunID); !errors.Is(err, ErrRunPaused) {
				t.Fatalf("resume bypassed pending approval: %v", err)
			}
			if _, err = ResumeApproval(ctx, restarted, paused.RunID, "wrong-id", true, ""); err == nil {
				t.Fatal("wrong approval accepted")
			}
			response, err := ResumeApproval(ctx, restarted, paused.RunID, "review-1", approved, "reviewed")
			if approved {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected continuation timeout: %v", err)
				}
				response, err = Resume(ctx, makeAgent(), paused.RunID)
			}
			if err != nil || response.RunID != paused.RunID || response.Reply != "finished" {
				t.Fatalf("response=%+v err=%v", response, err)
			}
			if _, err = Resume(ctx, makeAgent(), paused.RunID); err != nil {
				t.Fatal(err)
			}
			wantCalls := 0
			if approved {
				wantCalls = 1
			}
			wantModels := 2
			if approved {
				wantModels = 3
			}
			if toolCalls != wantCalls || modelCalls != wantModels || approvalCalls != 1 {
				t.Fatalf("tool=%d model=%d approval=%d", toolCalls, modelCalls, approvalCalls)
			}
		})
	}
}

type failingApprovalCheckpoint struct {
	flow.Checkpoint
	saves int
}

func (cp *failingApprovalCheckpoint) Save(ctx context.Context, run flow.Run) error {
	cp.saves++
	if cp.saves == 2 {
		return errors.New("approval storage unavailable")
	}
	return cp.Checkpoint.Save(ctx, run)
}
func TestDurableApprovalFailsClosedOnSaveError(t *testing.T) {
	cp := &failingApprovalCheckpoint{Checkpoint: flow.StoreCheckpoint(store.NewMemoryStore(), "failed-approval")}
	executed := 0
	a := newTestAgent(Name("save-failure"), WithCheckpoint(cp), WithApproval(func(context.Context, model.ToolCall) (ApprovalDecision, error) {
		return ApprovalDecision{Status: ApprovalPending, ID: "review"}, nil
	}), WithTool("publish", "", nil, func(context.Context, map[string]any) (string, error) { executed++; return "ok", nil }))
	fakeGen = func(ctx context.Context, opts model.Options, _ *model.Request) (*model.Response, error) {
		opts.ToolHandler(ctx, model.ToolCall{ID: "one", Name: "publish"})
		opts.ToolHandler(ctx, model.ToolCall{ID: "two", Name: "publish"})
		return &model.Response{Reply: "ignored refusal"}, nil
	}
	defer func() { fakeGen = nil }()
	if _, err := a.Ask(context.Background(), "publish"); err == nil || !strings.Contains(err.Error(), "approval storage unavailable") {
		t.Fatalf("save error lost: %v", err)
	}
	if executed != 0 {
		t.Fatal("tool executed without durable approval")
	}
}
