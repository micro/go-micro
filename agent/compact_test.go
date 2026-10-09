package agent

import (
	"go-micro.dev/v6/store"
	"testing"
)

func TestManualCompactionRetainsSearchableArchive(t *testing.T) {
	st := store.NewMemoryStore()
	memory := NewMemory(sessionStore(st, "memory", "one"), "history", 50)
	memory.Add("user", "remember the old marker")
	memory.Add("assistant", "acknowledged")
	memory.Add("user", "recent message")
	if err := memory.(MemoryCompactor).Compact(1); err != nil {
		t.Fatal(err)
	}
	history, err := LoadHistory(st, "memory", "one")
	if err != nil || len(history) != 2 || history[1].Content != "recent message" {
		t.Fatalf("history=%v err=%v", history, err)
	}
	matches, err := SearchHistory(st, "memory", "one", "old marker", 20)
	if err != nil || len(matches) == 0 {
		t.Fatalf("matches=%v err=%v", matches, err)
	}
	found := false
	for _, m := range matches {
		if m.Role == "user" && m.Content == "remember the old marker" {
			found = true
		}
	}
	if !found {
		t.Fatal("original message was lost")
	}
}
