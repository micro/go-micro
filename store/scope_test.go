package store

import "testing"

func TestScopeIsolatesTables(t *testing.T) {
	base := NewMemoryStore()

	a := Scope(base, "agent", "one")
	b := Scope(base, "agent", "two")

	if err := a.Write(&Record{Key: "history", Value: []byte("A")}); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := b.Write(&Record{Key: "history", Value: []byte("B")}); err != nil {
		t.Fatalf("write b: %v", err)
	}

	// Same key, different scopes — the values must not collide.
	recs, err := a.Read("history")
	if err != nil || len(recs) != 1 || string(recs[0].Value) != "A" {
		t.Fatalf("scope a read = %v %v", recs, err)
	}
	recs, err = b.Read("history")
	if err != nil || len(recs) != 1 || string(recs[0].Value) != "B" {
		t.Fatalf("scope b read = %v %v", recs, err)
	}

	// List is confined to the scope.
	keys, err := a.List()
	if err != nil || len(keys) != 1 {
		t.Fatalf("scope a list = %v %v, want 1 key", keys, err)
	}
}
