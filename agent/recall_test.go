package agent

import (
	"context"
	"errors"
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
