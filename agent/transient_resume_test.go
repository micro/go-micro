package agent

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestResumeTransientFailureAfterRestart(t *testing.T) {
	for _, failure := range []struct {
		name string
		err  error
	}{
		{"timeout", context.DeadlineExceeded}, {"rate_limited", testStatusError{code: 429}},
	} {
		for _, mode := range []string{"resume", "stream", "pending"} {
			t.Run(failure.name+"/"+mode, func(t *testing.T) {
				ctx := context.Background()
				st := store.NewMemoryStore()
				cp := flow.StoreCheckpoint(st, "recovery")
				modelCalls, toolCalls := 0, 0
				fakeGen = func(ctx context.Context, opts model.Options, req *model.Request) (*model.Response, error) {
					modelCalls++
					if req.Prompt != "charge once" {
						t.Fatalf("lost original request: %q", req.Prompt)
					}
					result := opts.ToolHandler(ctx, model.ToolCall{ID: "charge", Name: "charge", Input: map[string]any{"order": "42"}})
					if result.Content != "paid" {
						t.Fatalf("lost tool result: %+v", result)
					}
					if modelCalls == 1 {
						return nil, failure.err
					}
					return &model.Response{Reply: "recovered"}, nil
				}
				defer func() { fakeGen = nil }()
				makeAgent := func() *agentImpl {
					return newTestAgent(Name("recovery"), WithStore(st), WithCheckpoint(cp), WithTool("charge", "charge", nil, func(context.Context, map[string]any) (string, error) { toolCalls++; return "paid", nil }))
				}
				original := makeAgent()
				if _, err := original.Ask(ctx, "charge once"); err == nil {
					t.Fatal("expected provider failure")
				}
				runs, err := Pending(ctx, original)
				if err != nil || len(runs) != 1 || runs[0].Status != failure.name {
					t.Fatalf("pending: %+v %v", runs, err)
				}
				id := runs[0].ID
				restarted := makeAgent()
				var response *Response
				switch mode {
				case "resume":
					response, err = Resume(ctx, restarted, id)
				case "pending":
					var failedID string
					failedID, err = ResumePending(ctx, restarted)
					if failedID != "" {
						t.Fatalf("failed run: %s", failedID)
					}
					if err == nil {
						run, _, loadErr := cp.Load(ctx, id)
						if loadErr != nil {
							t.Fatal(loadErr)
						}
						response = new(Response)
						err = json.Unmarshal(run.State.Data, response)
					}
				case "stream":
					var stream AgentStream
					stream, err = ResumeStreamAsk(ctx, restarted, id)
					if err != nil {
						t.Fatal(err)
					}
					defer stream.Close()
					for {
						event, recvErr := stream.Recv()
						if recvErr == io.EOF {
							break
						}
						if recvErr != nil {
							err = recvErr
							break
						}
						if event.Type == StreamEventDone {
							response = event.Response
						}
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if response == nil || response.RunID != id || response.Reply != "recovered" {
					t.Fatalf("response: %+v", response)
				}
				if toolCalls != 1 || modelCalls != 2 {
					t.Fatalf("tool calls=%d, model calls=%d", toolCalls, modelCalls)
				}
				run, ok, err := cp.Load(ctx, id)
				if err != nil || !ok || run.Status != "done" {
					t.Fatalf("final checkpoint: %+v %v", run, err)
				}
			})
		}
	}
}
