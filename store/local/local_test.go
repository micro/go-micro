package local

import (
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kr/pretty"
	"github.com/micro/go-micro/v2/store"
)

func TestLocalReInit(t *testing.T) {
	s := NewStore(store.Table("aaa"))
	s.Init(store.Table(""))
	if len(s.Options().Table) > 0 {
		t.Error("Init didn't reinitialise the store")
	}
}

func TestLocalBasic(t *testing.T) {
	s := NewStore()
	s.Init()
	if err := s.(*localStore).deleteAll(); err != nil {
		t.Logf("Can't delete all: %v", err)
	}
	basictest(s, t)
}

func TestLocalTable(t *testing.T) {
	s := NewStore()
	s.Init(store.Table("some-Table"))
	if err := s.(*localStore).deleteAll(); err != nil {
		t.Logf("Can't delete all: %v", err)
	}
	basictest(s, t)
}

func TestLocalDatabase(t *testing.T) {
	s := NewStore()
	s.Init(store.Database("some-Database"))
	if err := s.(*localStore).deleteAll(); err != nil {
		t.Logf("Can't delete all: %v", err)
	}
	basictest(s, t)
}

func TestLocalDatabaseTable(t *testing.T) {
	s := NewStore()
	s.Init(store.Table("some-Table"), store.Database("some-Database"))
	if err := s.(*localStore).deleteAll(); err != nil {
		t.Logf("Can't delete all: %v", err)
	}
	basictest(s, t)
}

func basictest(s store.Store, t *testing.T) {
	t.Logf("Testing store %s, with options %# v\n", s.String(), pretty.Formatter(s.Options()))
	// Read and Write an expiring Record
	if err := s.Write(&store.Record{
		Key:    "Hello",
		Value:  []byte("World"),
		Expiry: time.Millisecond * 100,
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
			t.Errorf("Expected %s, got %s", "Hello", r[0].Key)
		}
		if string(r[0].Value) != "World" {
			t.Errorf("Expected %s, got %s", "World", r[0].Value)
		}
	}
	time.Sleep(time.Millisecond * 200)
	if _, err := s.Read("Hello"); err != store.ErrNotFound {
		t.Errorf("Expected %# v, got %# v", store.ErrNotFound, err)
	}

	// Write 3 records with various expiry and get with Table
	records := []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "foobar",
			Value:  []byte("foobarfoobar"),
			Expiry: time.Millisecond * 100,
		},
		&store.Record{
			Key:    "foobarbaz",
			Value:  []byte("foobarbazfoobarbaz"),
			Expiry: 2 * time.Millisecond * 100,
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 3 {
			t.Errorf("Expected 3 items, got %d", len(results))
			t.Logf("Table test: %v\n", spew.Sdump(results))
		}
	}
	time.Sleep(time.Millisecond * 100)
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
			t.Logf("Table test: %v\n", spew.Sdump(results))
		}
	}
	time.Sleep(time.Millisecond * 100)
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 1 {
			t.Errorf("Expected 1 item, got %d", len(results))
			t.Logf("Table test: %# v\n", spew.Sdump(results))
		}
	}
	if err := s.Delete("foo", func(d *store.DeleteOptions) {}); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expected 0 items, got %d (%# v)", len(results), spew.Sdump(results))
		}
	}

	// Write 3 records with various expiry and get with Suffix
	records = []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "barfoo",
			Value:  []byte("barfoobarfoo"),
			Expiry: time.Millisecond * 100,
		},
		&store.Record{
			Key:    "bazbarfoo",
			Value:  []byte("bazbarfoobazbarfoo"),
			Expiry: 2 * time.Millisecond * 100,
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 3 {
			t.Errorf("Expected 3 items, got %d", len(results))
			t.Logf("Table test: %v\n", spew.Sdump(results))
		}

	}
	time.Sleep(time.Millisecond * 100)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
			t.Logf("Table test: %v\n", spew.Sdump(results))
		}

	}
	time.Sleep(time.Millisecond * 100)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 1 {
			t.Errorf("Expected 1 item, got %d", len(results))
			t.Logf("Table test: %# v\n", spew.Sdump(results))
		}
	}
	if err := s.Delete("foo"); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expected 0 items, got %d (%# v)", len(results), spew.Sdump(results))
		}
	}

	// Test Table, Suffix and WriteOptions
	if err := s.Write(&store.Record{
		Key:   "foofoobarbar",
		Value: []byte("something"),
	}, store.WriteTTL(time.Millisecond*100)); err != nil {
		t.Error(err)
	}
	if err := s.Write(&store.Record{
		Key:   "foofoo",
		Value: []byte("something"),
	}, store.WriteExpiry(time.Now().Add(time.Millisecond*100))); err != nil {
		t.Error(err)
	}
	if err := s.Write(&store.Record{
		Key:   "barbar",
		Value: []byte("something"),
		// TTL has higher precedence than expiry
	}, store.WriteExpiry(time.Now().Add(time.Hour)), store.WriteTTL(time.Millisecond*100)); err != nil {
		t.Error(err)
	}
	if results, err := s.Read("foo", store.ReadPrefix(), store.ReadSuffix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 1 {
			t.Errorf("Expected 1 results, got %d: %# v", len(results), spew.Sdump(results))
		}
	}
	time.Sleep(time.Millisecond * 100)
	if results, err := s.List(); err != nil {
		t.Errorf("List failed: %s", err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expiry options were not effective, results :%v", spew.Sdump(results))
		}
	}
	s.Write(&store.Record{Key: "a", Value: []byte("a")})
	s.Write(&store.Record{Key: "aa", Value: []byte("aa")})
	s.Write(&store.Record{Key: "aaa", Value: []byte("aaa")})
	if results, err := s.Read("b", store.ReadPrefix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	}

	s.Init()
	if err := s.(*localStore).deleteAll(); err != nil {
		t.Logf("Can't delete all: %v", err)
	}
	for i := 0; i < 10; i++ {
		s.Write(&store.Record{
			Key:   fmt.Sprintf("a%d", i),
			Value: []byte{},
		})
	}
	if results, err := s.Read("a", store.ReadLimit(5), store.ReadPrefix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 5 {
			t.Error("Expected 5 results, got ", len(results))
		}
		if results[0].Key != "a0" {
			t.Errorf("Expected a0, got %s", results[0].Key)
		}
		if results[4].Key != "a4" {
			t.Errorf("Expected a4, got %s", results[4].Key)
		}
	}
	if results, err := s.Read("a", store.ReadLimit(30), store.ReadOffset(5), store.ReadPrefix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 5 {
			t.Error("Expected 5 results, got ", len(results))
		}
	}
}
