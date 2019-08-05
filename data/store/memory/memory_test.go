package memory

import (
	"testing"
	"time"

	"github.com/micro/go-micro/data/store"
)

func TestReadRecordExpire(t *testing.T) {
	s := NewStore()

	var (
		key    = "foo"
		expire = 100 * time.Millisecond
	)
	rec := &store.Record{
		Key:    key,
		Value:  nil,
		Expiry: expire,
	}
	s.Write(rec)

	rrec, err := s.Read(key)
	if err != nil {
		t.Fatal(err)
	}
	if rrec.Expiry >= expire {
		t.Fatal("expiry of read record is not changed")
	}

	time.Sleep(expire)

	if _, err := s.Read(key); err != store.ErrNotFound {
		t.Fatal("expire elapsed, but key still accessable")
	}
}
