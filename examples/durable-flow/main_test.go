package main

import (
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"testing"
)

func TestRestartableComposition(t *testing.T) {
	dir := t.TempDir()
	for _, action := range []string{"start", "status", "approve", "start"} {
		if err := run(dir, "one", action); err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}
	db, err := bolt.Open(filepath.Join(dir, "effects.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.View(func(tx *bolt.Tx) error {
		if got := tx.Bucket([]byte("effects")).Stats().KeyN; got != 2 {
			t.Errorf("effects=%d, want prepare + commit", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
