package chat

import (
	"context"
	"go-micro.dev/v6/agent/workspace"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/model"
)

func TestWorkspaceApproval(t *testing.T) {
	s := &session{output: io.Discard, approvals: make(chan approvalRequest)}
	decision, err := s.approve(context.Background(), model.ToolCall{Name: "workspace_exec"})
	if err != nil || decision.Status != agent.ApprovalDenied {
		t.Fatalf("noninteractive action: %+v %v", decision, err)
	}
	decision, err = s.approve(context.Background(), model.ToolCall{Name: "workspace_read"})
	if err != nil || decision.Status != agent.ApprovalApproved {
		t.Fatal("read should be allowed")
	}
	s.interactiveUI = true
	for _, allowed := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		done := make(chan agent.ApprovalDecision, 1)
		go func() { d, _ := s.approve(ctx, model.ToolCall{Name: "workspace_exec"}); done <- d }()
		select {
		case req := <-s.approvals:
			req.answer <- allowed
		case <-ctx.Done():
			t.Fatal("approval not requested")
		}
		d := <-done
		cancel()
		if (d.Status == agent.ApprovalApproved) != allowed {
			t.Fatalf("decision: %+v", d)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.approve(ctx, model.ToolCall{Name: "workspace_exec"}); err != context.Canceled {
		t.Fatalf("canceled approval: %v", err)
	}
}

func TestWorkspaceToolsThroughAgent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	s := testSession(t, func(ctx context.Context, r *model.Request, o model.Options) (*model.Response, error) {
		if !strings.Contains(r.SystemPrompt, "Project instructions") {
			t.Error("missing project instructions")
		}
		for _, call := range []model.ToolCall{
			{Name: "workspace_read", Input: map[string]any{"path": "note.txt"}},
			{Name: "workspace_edit", Input: map[string]any{"path": "note.txt", "old_text": "before", "new_text": "after"}},
			{Name: "workspace_search", Input: map[string]any{"query": "after"}},
		} {
			result := o.ToolHandler(ctx, call)
			if result.Refused != "" {
				t.Errorf("%s refused: %s", call.Name, result.Refused)
			}
		}
		return &model.Response{Reply: "done"}, nil
	})
	var err error
	s.workspace, err = workspace.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.yes = true
	s.instructions = "\nProject instructions: edit note.txt"
	if _, err = s.developmentAgent().Ask(context.Background(), "make the change"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "note.txt"))
	if err != nil || string(data) != "after" {
		t.Fatalf("edit: %q %v", data, err)
	}
}
