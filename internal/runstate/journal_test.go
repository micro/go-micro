package runstate

import (
	"context"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"testing"
)

func TestRejectFutureRecordSchema(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if err := j.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("runs"))
		if err != nil {
			return err
		}
		return b.Put([]byte("future"), []byte(`{"id":"future","schema_version":99}`))
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := j.Load(context.Background(), "future"); err == nil {
		t.Fatal("accepted future schema")
	}
	if _, err := j.List(context.Background()); err == nil {
		t.Fatal("silently omitted invalid run")
	}
}
