package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestSearchSessionsScopeAndRanking(t *testing.T) {
	st := store.NewMemoryStore()
	save := func(name, id string, state memoryState) {
		t.Helper()
		if err := sessionStore(st, name, id).Write(store.NewRecord("history", state)); err != nil {
			t.Fatal(err)
		}
	}
	save("assistant", "old", memoryState{Archive: []model.Message{{Role: "user", Content: "postgres migration uses batches"}, {Role: "assistant", Content: "postgres migration archived-only decision"}}, Messages: []model.Message{{Role: "system", Content: "postgres migration private instructions"}, {Role: "user", Content: "postgres migration uses batches"}}})
	save("assistant", "new", memoryState{Messages: []model.Message{{Role: "assistant", Content: "postgres is available"}, {Role: "assistant", Content: "postgres migration completed"}}})
	save("assistant", "excluded", memoryState{Messages: []model.Message{{Role: "user", Content: "postgres migration another user"}}})
	save("other-agent", "old", memoryState{Messages: []model.Message{{Role: "user", Content: "postgres migration another agent"}}})
	matches, err := SearchSessions(context.Background(), st, "assistant", []string{"new", "old", "old"}, "postgres migration", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 4 {
		t.Fatalf("matches: %+v", matches)
	}
	if matches[0].Session != "new" || matches[0].Message.Content != "postgres migration completed" || matches[1].Session != "old" || matches[2].Message.Content != "postgres migration archived-only decision" || matches[3].Message.Content != "postgres is available" {
		t.Fatalf("ranking or scope: %+v", matches)
	}
	limited, err := SearchSessions(context.Background(), st, "assistant", []string{"new", "old"}, "postgres migration", 1)
	if err != nil || len(limited) != 1 || limited[0] != matches[0] {
		t.Fatalf("limit: %+v %v", limited, err)
	}
	none, err := SearchSessions(context.Background(), st, "assistant", nil, "postgres", 10)
	if err != nil || len(none) != 0 {
		t.Fatalf("empty scope: %+v %v", none, err)
	}
}

type failedRecallStore struct{ store.Store }

func (s failedRecallStore) Read(string, ...store.ReadOption) ([]*store.Record, error) {
	return nil, errors.New("storage unavailable")
}

func TestSearchSessionsErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SearchSessions(ctx, store.NewMemoryStore(), "agent", []string{"one"}, "postgres", 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := SearchSessions(context.Background(), failedRecallStore{store.NewMemoryStore()}, "agent", []string{"one"}, "postgres", 10); err == nil {
		t.Fatal("storage failure hidden")
	}
}

func TestSearchSessionsOmitsActiveTurnBeforeRanking(t *testing.T) {
	for _, retrieval := range []bool{false, true} {
		t.Run(fmt.Sprint(retrieval), func(t *testing.T) {
			st := store.NewMemoryStore()
			old := memoryState{Messages: []model.Message{{Role: "assistant", Content: "postgres decision was to use batches"}}}
			if err := sessionStore(st, "recaller", "earlier").Write(store.NewRecord("history", old)); err != nil {
				t.Fatal(err)
			}
			var matches []HistoryMatch
			fakeGen = func(ctx context.Context, opts model.Options, _ *model.Request) (*model.Response, error) {
				result := opts.ToolHandler(ctx, model.ToolCall{ID: "recall", Name: "recall", Input: map[string]any{}})
				if result.Content != "ok" {
					t.Errorf("recall: %+v", result)
				}
				return &model.Response{Reply: "done"}, nil
			}
			defer func() { fakeGen = nil }()
			opts := []Option{Name("recaller"), Session("current"), WithStore(st), WithTool("recall", "recall", nil, func(ctx context.Context, _ map[string]any) (string, error) {
				var err error
				matches, err = SearchSessions(ctx, st, "recaller", []string{"current", "earlier"}, "postgres decision", 1)
				return "ok", err
			})}
			if retrieval {
				opts = append(opts, RetrievalMemory(10))
			} else {
				opts = append(opts, CompactMemory(10, 5))
			}
			a := newTestAgent(opts...)
			if _, err := a.Ask(context.Background(), "what was the postgres decision"); err != nil {
				t.Fatal(err)
			}
			if len(matches) != 1 || matches[0].Session != "earlier" {
				t.Fatalf("active question crowded out history: %+v", matches)
			}
			// Once the turn finishes, explicit searches can inspect the stored question.
			after, err := SearchSessions(context.Background(), st, "recaller", []string{"current"}, "postgres decision", 10)
			if err != nil || len(after) != 1 {
				t.Fatalf("completed turn was removed: %+v %v", after, err)
			}
			// An identical older question must remain searchable on the next turn.
			if _, err := a.Ask(context.Background(), "what was the postgres decision"); err != nil {
				t.Fatal(err)
			}
			if len(matches) != 1 || matches[0].Session != "current" {
				t.Fatalf("older current-session message lost: %+v", matches)
			}
		})
	}
}
