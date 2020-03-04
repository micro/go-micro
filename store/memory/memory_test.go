package memory

import (
	"testing"
	"time"

	"github.com/micro/go-micro/v2/store"
)

func TestMemoryBasic(t *testing.T) {
	s := NewStore()
	s.Init()
	basictest(s, t)
}

func TestMemoryPrefix(t *testing.T) {
	s := NewStore()
	s.Init(store.Prefix("some-prefix"))
	basictest(s, t)
}

func basictest(s store.Store, t *testing.T) {
	// Read and Write an expiring Record
	if err := s.Write(&store.Record{
		Key:    "Hello",
		Value:  []byte("World"),
		Expiry: time.Second,
	}); err != nil {
		t.Error(err)
	}
	if r, err := s.Read("Hello"); err != nil {
		t.Error(err)
	} else {
		if len(r) != 1 {
			t.Error("Read returned multiple records")
		}
		if r[0].Key != "Hello" {
			t.Errorf("Expected %s, got %v", "Hello", r[0].Key)
		}
		if string(r[0].Value) != "World" {
			t.Errorf("Expected %s, got %v", "World", r[0].Value)
		}
	}
	time.Sleep(time.Second * 2)
	if _, err := s.Read("Hello"); err != store.ErrNotFound {
		t.Errorf("Expected %v, got %v", store.ErrNotFound, err)
	}

	// Write 3 records with various expiry and get with prefix
	records := []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "foobar",
			Value:  []byte("foobarfoobar"),
			Expiry: time.Second,
		},
		&store.Record{
			Key:    "foobarbaz",
			Value:  []byte("foobarbazfoobarbaz"),
			Expiry: 2 * time.Second,
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Error(err)
		}
	}
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 3 {
			t.Errorf("Expected 3 items, got %d", len(results))
		}
	}
}
